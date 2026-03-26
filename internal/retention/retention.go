package retention

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Cleanup은 보관 기간이 지난 날짜별 JSONL 로그 파일을 삭제한다.
//
// YYYY-MM-DD.jsonl 형식의 파일만 삭제 대상으로 간주하며,
// 그 외 파일은 무시한다.
func Cleanup(logDir string, retentionDays int, now time.Time) error {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read log dir: %w", err)
	}

	cutoff := now.AddDate(0, 0, -retentionDays)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".jsonl" {
			continue
		}
		base := name[:len(name)-len(".jsonl")]
		day, err := time.Parse("2006-01-02", base)
		if err != nil {
			continue
		}
		if day.Before(cutoff) {
			if err := os.Remove(filepath.Join(logDir, name)); err != nil {
				return fmt.Errorf("remove old log file %s: %w", name, err)
			}
		}
	}

	return nil
}
