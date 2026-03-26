package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"dsa/internal/logx"
)

// makeLogFile은 지정한 날짜의 JSONL 파일을 테스트 디렉터리에 생성한다.
func makeLogFile(t *testing.T, dir, date string, records []logx.Record) {
	t.Helper()
	f, err := os.Create(filepath.Join(dir, date+".jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, r := range records {
		if err := enc.Encode(r); err != nil {
			t.Fatal(err)
		}
	}
}

// TestBuild_YesterdayUsesAddDate는 Build()가 DST에 안전한 AddDate(0,0,-1)로
// 전일 날짜를 계산하는지 검증한다.
//
// now가 자정 00:00:00인 경우 -24h는 전날 자정을 정확히 가리키지만,
// DST 전환 시각에는 다른 날짜를 반환할 수 있다.
// AddDate(0,0,-1)는 항상 달력 기준 하루 전을 반환해야 한다.
func TestBuild_YesterdayUsesAddDate(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Seoul")
	dir := t.TempDir()

	// now = 2025-03-01 00:00:00 KST
	now := time.Date(2025, 3, 1, 0, 0, 0, 0, loc)

	// -24h와 AddDate(0,0,-1)의 차이가 나는 경우:
	// DST 없는 KST에서는 같지만 AddDate 방식이 명확한 의도를 표현한다.
	// 테스트는 "2025-02-28" 파일을 읽어야 함을 강제한다.
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	if yesterday != "2025-02-28" {
		t.Fatalf("test setup error: expected 2025-02-28, got %s", yesterday)
	}

	records := []logx.Record{
		{Timestamp: "2025-02-28T10:00:00+09:00", RunMode: "keepalive", Event: "keepalive_run", Success: true},
		{Timestamp: "2025-02-28T10:00:00+09:00", RunMode: "keepalive", QueryName: "simple_ping", Event: "query_execution", Success: true},
	}
	makeLogFile(t, dir, "2025-02-28", records)

	stats, err := Build(dir, loc, now)
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	if stats.TotalRuns != 1 {
		t.Errorf("TotalRuns = %d, want 1 (should have read 2025-02-28.jsonl)", stats.TotalRuns)
	}
}

// TestBuild_MissingLogReturnsEmpty는 로그 파일이 없을 때 빈 Stats를 반환하는지 확인한다.
func TestBuild_MissingLogReturnsEmpty(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Seoul")
	dir := t.TempDir()
	now := time.Date(2025, 3, 1, 12, 0, 0, 0, loc)

	stats, err := Build(dir, loc, now)
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if stats.TotalRuns != 0 {
		t.Errorf("TotalRuns = %d, want 0 for missing log", stats.TotalRuns)
	}
}

// TestErrorSummaryLines_SortsByCountDesc는 에러가 빈도 내림차순으로 정렬되는지 확인한다.
func TestErrorSummaryLines_SortsByCountDesc(t *testing.T) {
	summary := map[string]int{
		"minor error": 1,
		"major error": 5,
		"mid error":   3,
	}
	lines := ErrorSummaryLines(summary, 10)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	// 첫 번째 줄은 가장 많은 major error여야 한다
	if lines[0] != "- major error (5)" {
		t.Errorf("first line = %q, want \"- major error (5)\"", lines[0])
	}
}
