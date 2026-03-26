package job

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dsa/internal/config"
	"dsa/internal/db"
	"dsa/internal/logx"
	"dsa/internal/notifier"
	"dsa/internal/report"
	"dsa/internal/retention"
)

// Run은 설정된 실행 모드에 맞는 작업을 선택해서 수행한다.
//
// 지원 모드:
// - keepalive
// - daily-report
func Run(ctx context.Context, cfg config.Config, logger *logx.Logger) error {
	location, err := time.LoadLocation(cfg.AppTimezone)
	if err != nil {
		return fmt.Errorf("load location: %w", err)
	}

	switch cfg.RunMode {
	case config.RunModeKeepalive:
		return runKeepalive(ctx, cfg, logger, location)
	case config.RunModeDailyReport:
		return runDailyReport(ctx, cfg, logger, location)
	default:
		return fmt.Errorf("unsupported run mode: %s", cfg.RunMode)
	}
}

// runKeepalive는 DB 유휴 방지 작업 1회를 수행한다.
//
// 처리 순서:
// 1. 3개의 탐침 쿼리를 고정 순서대로 실행한다.
// 2. 실패한 쿼리는 MaxRetryCount만큼 재시도한다.
// 3. 쿼리별 로그를 남긴다.
// 4. 1회 실행 요약 로그를 남긴다.
// 5. 보관 기간이 지난 로그를 삭제한다.
// 6. 3개 쿼리가 모두 실패했을 때만 카카오워크 장애 알림을 보낸다.
func runKeepalive(ctx context.Context, cfg config.Config, logger *logx.Logger, location *time.Location) error {
	now := time.Now().In(location)
	queries := db.DefaultQueries()
	results := make([]db.QueryResult, 0, len(queries))

	for _, query := range queries {
		result := executeWithRetry(ctx, cfg, query)
		results = append(results, result)

		if err := logger.Write(logx.Record{
			Timestamp:  now.Format(time.RFC3339),
			RunMode:    cfg.RunMode,
			QueryName:  result.Name,
			SQL:        result.SQL,
			Attempt:    result.Attempt,
			Success:    result.Success,
			DurationMS: result.DurationMS,
			Error:      result.Error,
			Event:      "query_execution",
		}); err != nil {
			return err
		}
	}

	anySuccess := false
	for _, result := range results {
		if result.Success {
			anySuccess = true
			break
		}
	}

	if err := logger.Write(logx.Record{
		Timestamp: now.Format(time.RFC3339),
		RunMode:   cfg.RunMode,
		Success:   anySuccess,
		Event:     "keepalive_run",
	}); err != nil {
		return err
	}

	if err := retention.Cleanup(cfg.LogDir, cfg.LogRetentionDays, now); err != nil {
		return err
	}

	if anySuccess {
		return nil
	}

	// 실패한 쿼리명 수집
	failedNames := make([]string, 0, len(results))
	for _, r := range results {
		failedNames = append(failedNames, r.Name)
	}
	lastError := results[len(results)-1].Error

	blocks := []any{
		notifier.Header("[DB Keepalive 실패]", "red"),
		notifier.Description("발생 시각", now.Format("2006-01-02 15:04:05 MST")),
		notifier.Description("DB", cfg.DBLabel),
		notifier.Description("실패 쿼리", strings.Join(failedNames, ", ")),
		notifier.DescriptionRed("마지막 에러", lastError),
		notifier.Description("재시도", fmt.Sprintf("%d회", cfg.MaxRetryCount)),
		notifier.Divider(),
	}

	client := notifier.New(cfg.KakaoWorkWebhookURL, cfg.HTTPTimeout)
	sendCtx, cancel := context.WithTimeout(ctx, cfg.HTTPTimeout)
	defer cancel()

	if err := client.Send(sendCtx, "[DB Keepalive 실패] "+cfg.DBLabel, blocks); err != nil {
		return err
	}

	return logger.Write(logx.Record{
		Timestamp: now.Format(time.RFC3339),
		RunMode:   cfg.RunMode,
		Success:   false,
		Event:     "webhook_notification",
	})
}

// runDailyReport는 전일 로그를 집계해서 일일 보고를 전송한다.
//
// 집계 범위는 "최근 24시간"이 아니라 APP_TIMEZONE 기준의 "전일 00:00:00~23:59:59"이다.
func runDailyReport(ctx context.Context, cfg config.Config, logger *logx.Logger, location *time.Location) error {
	now := time.Now().In(location)
	stats, err := report.Build(cfg.LogDir, location, now)
	if err != nil {
		return err
	}

	yesterday := now.AddDate(0, 0, -1)
	period := fmt.Sprintf(
		"%s 00:00 ~ 23:59 %s",
		yesterday.Format("2006-01-02"),
		yesterday.Format("MST"),
	)

	lastFailure := "없음"
	if stats.LastFailureAt != "" {
		lastFailure = stats.LastFailureAt
	}

	queryTable := buildQueryTable(stats)
	errorText := buildErrorText(stats)

	blocks := []any{
		notifier.Header("[DB Keepalive 일일 보고]", "blue"),
		notifier.Description("집계 기간", period),
		notifier.Description("실행 횟수", fmt.Sprintf("%d회", stats.TotalRuns)),
		notifier.Description("마지막 실패", lastFailure),
		notifier.Divider(),
		notifier.Text(queryTable),
		notifier.Divider(),
		notifier.Text(errorText),
	}

	client := notifier.New(cfg.KakaoWorkWebhookURL, cfg.HTTPTimeout)
	sendCtx, cancel := context.WithTimeout(ctx, cfg.HTTPTimeout)
	defer cancel()

	if err := client.Send(sendCtx, "[DB Keepalive 일일 보고]", blocks); err != nil {
		return err
	}

	if err := retention.Cleanup(cfg.LogDir, cfg.LogRetentionDays, now); err != nil {
		return err
	}

	return logger.Write(logx.Record{
		Timestamp: now.Format(time.RFC3339),
		RunMode:   cfg.RunMode,
		Success:   true,
		Event:     "daily_report_sent",
	})
}

// buildQueryTable은 쿼리별 성공/실패 통계를 text 블록용 문자열로 만든다.
func buildQueryTable(stats report.Stats) string {
	var sb strings.Builder
	sb.WriteString("쿼리 결과")
	for _, q := range db.DefaultQueries() {
		sb.WriteString(fmt.Sprintf("\n%s: 성공 %d / 실패 %d",
			q.Name,
			stats.QuerySuccess[q.Name],
			stats.QueryFailure[q.Name],
		))
	}
	return sb.String()
}

// buildErrorText는 에러 요약을 text 블록용 문자열로 만든다.
func buildErrorText(stats report.Stats) string {
	lines := report.ErrorSummaryLines(stats.ErrorSummary, 3)
	return "주요 에러\n" + strings.Join(lines, "\n")
}

// executeWithRetry는 쿼리 1개를 재시도 정책과 함께 실행한다.
//
// 예시:
// - MaxRetryCount = 0 이면 최초 1회만 실행
// - MaxRetryCount = 1 이면 최초 실행 후 1회 재시도
func executeWithRetry(ctx context.Context, cfg config.Config, query db.QueryDefinition) db.QueryResult {
	var latest db.QueryResult
	for attempt := 1; attempt <= cfg.MaxRetryCount+1; attempt++ {
		queryCtx, cancel := context.WithTimeout(ctx, cfg.QueryTimeout)
		result := db.RunQuery(queryCtx, cfg.DatabaseURL, query)
		cancel()

		result.Attempt = attempt
		latest = result
		if result.Success {
			return result
		}
	}
	return latest
}
