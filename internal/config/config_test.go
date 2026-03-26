package config

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMerge_ZeroIntFromFileOverridesDefault는 JSON 설정 파일에서
// 명시적으로 0을 지정했을 때 코드 기본값을 덮어쓰는지 검증한다.
//
// 재현 시나리오: 재시도 없이 실행하려고 "max_retry_count": 0 설정 시,
// 기존 merge()는 0을 "설정 없음"으로 판단해 기본값(1)을 유지한다.
func TestMerge_ZeroIntFromFileOverridesDefault(t *testing.T) {
	dst := defaultConfig("/tmp")
	if dst.MaxRetryCount != 1 {
		t.Fatalf("precondition: default MaxRetryCount = %d, want 1", dst.MaxRetryCount)
	}

	// JSON에서 max_retry_count = 0 을 명시한 fileConfig
	src := fileConfig{MaxRetryCount: intPtr(0)}
	merge(&dst, src)

	if dst.MaxRetryCount != 0 {
		t.Errorf("MaxRetryCount = %d, want 0 (explicit zero from file should override default)", dst.MaxRetryCount)
	}
}

// TestMerge_ZeroDailyReportHourOverridesDefault는 daily_report_hour=0 (자정)이
// 기본값(9)을 올바르게 덮어쓰는지 검증한다.
func TestMerge_ZeroDailyReportHourOverridesDefault(t *testing.T) {
	dst := defaultConfig("/tmp")

	src := fileConfig{DailyReportHour: intPtr(0)}
	merge(&dst, src)

	if dst.DailyReportHour != 0 {
		t.Errorf("DailyReportHour = %d, want 0 (midnight should be valid)", dst.DailyReportHour)
	}
}

// TestMerge_NilIntFieldKeepsDefault는 JSON에 없는 int 필드는 기본값을 유지하는지 확인한다.
func TestMerge_NilIntFieldKeepsDefault(t *testing.T) {
	dst := defaultConfig("/tmp")

	// MaxRetryCount 없이 merge
	src := fileConfig{}
	merge(&dst, src)

	if dst.MaxRetryCount != 1 {
		t.Errorf("MaxRetryCount = %d, want 1 (nil should keep default)", dst.MaxRetryCount)
	}
}

// TestLoadFromFile_ZeroValueParsed는 JSON 파일에서 0이 올바르게 파싱되는지 확인한다.
func TestLoadFromFile_ZeroValueParsed(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dsa.config.json")

	content := map[string]any{
		"database_url":          "postgres://test",
		"kakaowork_webhook_url": "https://test.com",
		"max_retry_count":       0,
		"daily_report_hour":     0,
	}
	raw, _ := json.Marshal(content)
	if err := os.WriteFile(cfgPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	fc, err := loadFromFile(cfgPath)
	if err != nil {
		t.Fatalf("loadFromFile() error: %v", err)
	}
	if fc.MaxRetryCount == nil || *fc.MaxRetryCount != 0 {
		t.Errorf("MaxRetryCount = %v, want pointer to 0", fc.MaxRetryCount)
	}
	if fc.DailyReportHour == nil || *fc.DailyReportHour != 0 {
		t.Errorf("DailyReportHour = %v, want pointer to 0", fc.DailyReportHour)
	}
}

// TestGetEnvInt_InvalidValueLogsWarning은 환경변수에 정수로 파싱 불가한 값이
// 들어왔을 때 slog.Warn 경고가 출력되는지 검증한다.
func TestGetEnvInt_InvalidValueLogsWarning(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(oldLogger)

	t.Setenv("DSA_TEST_INT", "notanumber")
	result := getEnvInt("DSA_TEST_INT", 42)

	if result != 42 {
		t.Errorf("result = %d, want 42 (fallback on parse error)", result)
	}
	if !strings.Contains(buf.String(), "DSA_TEST_INT") {
		t.Errorf("expected warning log containing key name, got: %q", buf.String())
	}
}

func intPtr(v int) *int { return &v }
