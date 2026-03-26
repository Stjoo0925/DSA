package job

import (
	"context"
	"fmt"
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

	client := notifier.New(cfg.KakaoWorkWebhookURL, cfg.HTTPTimeout)
	lines := []string{
		"발생 시각: " + now.Format("2006-01-02 15:04:05 MST"),
		"DB: " + cfg.DBLabel,
	}
	for _, result := range results {
		lines = append(lines, fmt.Sprintf("%s: fail", result.Name))
	}
	lines = append(lines, "마지막 에러: "+results[len(results)-1].Error)
	lines = append(lines, fmt.Sprintf("재시도 횟수: %d", cfg.MaxRetryCount))

	sendCtx, cancel := context.WithTimeout(ctx, cfg.HTTPTimeout)
	defer cancel()

	if err := client.Send(sendCtx, "[DB Keepalive 실패]", lines); err != nil {
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

	lines := []string{
		fmt.Sprintf("집계 기간: %s ~ %s", stats.PeriodStart.Format("2006-01-02 15:04:05 MST"), stats.PeriodEnd.Format("2006-01-02 15:04:05 MST")),
		fmt.Sprintf("총 실행 횟수: %d", stats.TotalRuns),
		fmt.Sprintf("simple_ping: success %d / fail %d", stats.QuerySuccess["simple_ping"], stats.QueryFailure["simple_ping"]),
		fmt.Sprintf("table_limit_probe: success %d / fail %d", stats.QuerySuccess["table_limit_probe"], stats.QueryFailure["table_limit_probe"]),
		fmt.Sprintf("full_table_probe: success %d / fail %d", stats.QuerySuccess["full_table_probe"], stats.QueryFailure["full_table_probe"]),
	}

	if stats.LastFailureAt == "" {
		lines = append(lines, "마지막 실패 시각: 없음")
	} else {
		lines = append(lines, "마지막 실패 시각: "+stats.LastFailureAt)
	}

	lines = append(lines, "주요 에러 요약:")
	lines = append(lines, report.ErrorSummaryLines(stats.ErrorSummary, 3)...)

	client := notifier.New(cfg.KakaoWorkWebhookURL, cfg.HTTPTimeout)
	sendCtx, cancel := context.WithTimeout(ctx, cfg.HTTPTimeout)
	defer cancel()

	if err := client.Send(sendCtx, "[DB Keepalive 일일 보고]", lines); err != nil {
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
