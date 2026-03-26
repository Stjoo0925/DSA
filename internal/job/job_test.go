package job

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"dsa/internal/config"
	"dsa/internal/db"
	"dsa/internal/logx"
	"dsa/internal/report"
)

// newTestLogger는 테스트용 임시 디렉터리에 쓰는 로거를 생성한다.
func newTestLogger(t *testing.T) *logx.Logger {
	t.Helper()
	return logx.New(t.TempDir(), slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelError})))
}

// --- buildQueryTable ---

func TestBuildQueryTable_ContainsAllQueryNames(t *testing.T) {
	stats := report.Stats{
		QuerySuccess: map[string]int{"simple_ping": 24, "table_limit_probe": 22, "full_table_probe": 20},
		QueryFailure: map[string]int{"simple_ping": 0, "table_limit_probe": 2, "full_table_probe": 4},
	}
	result := buildQueryTable(stats)

	for _, name := range []string{"simple_ping", "table_limit_probe", "full_table_probe"} {
		if !strings.Contains(result, name) {
			t.Errorf("buildQueryTable() missing %q in output:\n%s", name, result)
		}
	}
}

func TestBuildQueryTable_ShowsSuccessAndFailureCounts(t *testing.T) {
	stats := report.Stats{
		QuerySuccess: map[string]int{"simple_ping": 10},
		QueryFailure: map[string]int{"simple_ping": 3},
	}
	result := buildQueryTable(stats)

	if !strings.Contains(result, "성공 10") {
		t.Errorf("buildQueryTable() should contain '성공 10', got:\n%s", result)
	}
	if !strings.Contains(result, "실패 3") {
		t.Errorf("buildQueryTable() should contain '실패 3', got:\n%s", result)
	}
}

func TestBuildQueryTable_StartsWithHeader(t *testing.T) {
	stats := report.Stats{
		QuerySuccess: map[string]int{},
		QueryFailure: map[string]int{},
	}
	result := buildQueryTable(stats)

	if !strings.HasPrefix(result, "쿼리 결과") {
		t.Errorf("buildQueryTable() should start with '쿼리 결과', got:\n%s", result)
	}
}

// --- buildErrorText ---

func TestBuildErrorText_NoErrorsShowsNone(t *testing.T) {
	stats := report.Stats{ErrorSummary: map[string]int{}}
	result := buildErrorText(stats)

	if !strings.Contains(result, "없음") {
		t.Errorf("buildErrorText() with no errors should contain '없음', got:\n%s", result)
	}
}

func TestBuildErrorText_ShowsErrorMessages(t *testing.T) {
	stats := report.Stats{ErrorSummary: map[string]int{"connection refused": 2}}
	result := buildErrorText(stats)

	if !strings.Contains(result, "connection refused") {
		t.Errorf("buildErrorText() should contain error message, got:\n%s", result)
	}
	if !strings.HasPrefix(result, "주요 에러") {
		t.Errorf("buildErrorText() should start with '주요 에러', got:\n%s", result)
	}
}

// --- executeWithRetry ---

func TestExecuteWithRetry_MaxRetry0MeansOneAttempt(t *testing.T) {
	cfg := config.Config{
		DatabaseURL:      "postgres://invalid-host-that-does-not-exist:9999/db",
		MaxRetryCount:    0,
		QueryTimeout:     2 * time.Second,
	}
	query := db.DefaultQueries()[0]

	result := executeWithRetry(context.Background(), cfg, query)

	if result.Attempt != 1 {
		t.Errorf("executeWithRetry() MaxRetryCount=0 Attempt = %d, want 1", result.Attempt)
	}
	if result.Success {
		t.Error("executeWithRetry() with invalid URL should not succeed")
	}
}

func TestExecuteWithRetry_MaxRetry2MeansThreeAttempts(t *testing.T) {
	cfg := config.Config{
		DatabaseURL:   "postgres://invalid-host-that-does-not-exist:9999/db",
		MaxRetryCount: 2,
		QueryTimeout:  2 * time.Second,
	}
	query := db.DefaultQueries()[0]

	result := executeWithRetry(context.Background(), cfg, query)

	if result.Attempt != 3 {
		t.Errorf("executeWithRetry() MaxRetryCount=2 Attempt = %d, want 3", result.Attempt)
	}
}

func TestExecuteWithRetry_ReturnsLastAttemptResult(t *testing.T) {
	cfg := config.Config{
		DatabaseURL:   "postgres://invalid-host-that-does-not-exist:9999/db",
		MaxRetryCount: 1,
		QueryTimeout:  2 * time.Second,
	}
	query := db.QueryDefinition{Name: "test_query", SQL: "SELECT 1"}

	result := executeWithRetry(context.Background(), cfg, query)

	if result.Name != "test_query" {
		t.Errorf("executeWithRetry() Name = %q, want %q", result.Name, "test_query")
	}
}

// --- Run ---

func TestRun_UnsupportedModeReturnsError(t *testing.T) {
	cfg := config.Config{
		RunMode:     "unsupported-mode",
		AppTimezone: "Asia/Seoul",
	}
	logger := newTestLogger(t)

	err := Run(context.Background(), cfg, logger)
	if err == nil {
		t.Fatal("Run() unsupported mode should return error")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("error should mention 'unsupported', got: %v", err)
	}
}

func TestRun_InvalidTimezoneReturnsError(t *testing.T) {
	cfg := config.Config{
		RunMode:     config.RunModeKeepalive,
		AppTimezone: "Invalid/Timezone",
	}
	logger := newTestLogger(t)

	err := Run(context.Background(), cfg, logger)
	if err == nil {
		t.Fatal("Run() invalid timezone should return error")
	}
}
