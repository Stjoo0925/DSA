package logx

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestWrite_UsesRecordTimestampForFilename는 Write()가 시스템 시계가 아닌
// record.Timestamp를 기준으로 로그 파일명을 결정하는지 검증한다.
//
// 재현 시나리오: 서버가 UTC 시간대일 때, 한국 시간 기준 다음 날 00:30에 실행된
// keepalive 기록은 KST 날짜 파일에 저장되어야 한다.
func TestWrite_UsesRecordTimestampForFilename(t *testing.T) {
	dir := t.TempDir()
	logger := New(dir, slog.Default())

	// 과거 특정 날짜의 타임스탬프를 가진 레코드 — 오늘 날짜와 다름
	record := Record{
		Timestamp: "2020-01-15T10:00:00+09:00",
		RunMode:   "keepalive",
		Event:     "test_event",
		Success:   true,
	}

	if err := logger.Write(record); err != nil {
		t.Fatalf("Write() unexpected error: %v", err)
	}

	// 레코드 타임스탬프 날짜 파일이 생성되어야 한다
	expectedFile := filepath.Join(dir, "2020-01-15.jsonl")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("expected log file %q to exist, but it does not", expectedFile)
	}

	// 오늘 날짜 파일은 생성되지 않아야 한다
	todayFile := filepath.Join(dir, time.Now().Format("2006-01-02")+".jsonl")
	if _, err := os.Stat(todayFile); !os.IsNotExist(err) {
		t.Errorf("today's log file %q should not exist when record has a different timestamp", todayFile)
	}
}
