package logx

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// Logger는 구조화된 실행 기록을 날짜별 JSONL 파일로 저장한다.
type Logger struct {
	dir    string
	logger *slog.Logger
}

// Record는 디스크에 저장되는 실행 이벤트 1건의 스키마다.
//
// JSONL 파일의 한 줄에는 Record 1개가 JSON 형태로 저장된다.
type Record struct {
	Timestamp  string `json:"timestamp"`
	RunMode    string `json:"run_mode"`
	QueryName  string `json:"query_name,omitempty"`
	SQL        string `json:"sql,omitempty"`
	Attempt    int    `json:"attempt,omitempty"`
	Success    bool   `json:"success"`
	DurationMS int64  `json:"duration_ms,omitempty"`
	Error      string `json:"error,omitempty"`
	Event      string `json:"event"`
}

// New는 지정한 디렉터리에 JSONL 로그를 쓰는 로거를 생성한다.
func New(dir string, logger *slog.Logger) *Logger {
	return &Logger{dir: dir, logger: logger}
}

// Write는 오늘 날짜 기준 JSONL 파일에 Record 1건을 추가한다.
//
// 파일명 형식:
// - YYYY-MM-DD.jsonl
func (l *Logger) Write(record Record) error {
	if err := os.MkdirAll(l.dir, 0o755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}

	ts, err := time.Parse(time.RFC3339, record.Timestamp)
	if err != nil {
		ts = time.Now()
	}
	filePath := filepath.Join(l.dir, ts.Format("2006-01-02")+".jsonl")
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer file.Close()

	payload, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal log record: %w", err)
	}

	if _, err := file.Write(append(payload, '\n')); err != nil {
		return fmt.Errorf("write log record: %w", err)
	}

	l.logger.Info("log record written", "event", record.Event, "run_mode", record.RunMode, "success", record.Success)
	return nil
}
