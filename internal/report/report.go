package report

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"dsa/internal/logx"
)

// Stats는 일일 보고 메시지에 필요한 집계 결과를 담는다.
type Stats struct {
	TotalRuns     int
	QuerySuccess  map[string]int
	QueryFailure  map[string]int
	LastFailureAt string
	ErrorSummary  map[string]int
	PeriodStart   time.Time
	PeriodEnd     time.Time
}

// Build는 전일 JSONL 로그 파일을 읽어서 일일 보고용 통계를 만든다.
//
// 집계 규칙:
// - TotalRuns는 쿼리 개수가 아니라 keepalive 실행 횟수를 센다.
// - QuerySuccess / QueryFailure는 쿼리별 최종 결과를 센다.
// - LastFailureAt은 마지막 실패 쿼리 시각을 저장한다.
// - ErrorSummary는 동일한 에러 메시지를 묶어서 개수를 센다.
func Build(logDir string, location *time.Location, now time.Time) (Stats, error) {
	dayTime := now.In(location).Add(-24 * time.Hour)
	day := dayTime.Format("2006-01-02")
	filePath := filepath.Join(logDir, day+".jsonl")

	empty := Stats{
		QuerySuccess: make(map[string]int),
		QueryFailure: make(map[string]int),
		ErrorSummary: make(map[string]int),
		PeriodStart:  time.Date(dayTime.Year(), dayTime.Month(), dayTime.Day(), 0, 0, 0, 0, location),
		PeriodEnd:    time.Date(dayTime.Year(), dayTime.Month(), dayTime.Day(), 23, 59, 59, 0, location),
	}

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return empty, nil
		}
		return Stats{}, fmt.Errorf("open report log file: %w", err)
	}
	defer file.Close()

	stats := empty

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record logx.Record
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return Stats{}, fmt.Errorf("decode report record: %w", err)
		}
		if record.Event == "keepalive_run" {
			stats.TotalRuns++
			continue
		}
		if record.Event != "query_execution" {
			continue
		}
		if record.Success {
			stats.QuerySuccess[record.QueryName]++
			continue
		}

		stats.QueryFailure[record.QueryName]++
		stats.LastFailureAt = record.Timestamp
		if record.Error != "" {
			stats.ErrorSummary[record.Error]++
		}
	}

	if err := scanner.Err(); err != nil {
		return Stats{}, fmt.Errorf("scan report file: %w", err)
	}

	return stats, nil
}

// ErrorSummaryLines는 에러 메시지 집계를 사람이 읽기 쉬운 문자열 목록으로 바꾼다.
//
// 정렬 기준:
// 1. 개수 내림차순
// 2. 메시지 오름차순
func ErrorSummaryLines(summary map[string]int, limit int) []string {
	type pair struct {
		Message string
		Count   int
	}
	pairs := make([]pair, 0, len(summary))
	for message, count := range summary {
		pairs = append(pairs, pair{Message: message, Count: count})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Count == pairs[j].Count {
			return pairs[i].Message < pairs[j].Message
		}
		return pairs[i].Count > pairs[j].Count
	})

	lines := make([]string, 0, limit)
	for i, item := range pairs {
		if i >= limit {
			break
		}
		lines = append(lines, fmt.Sprintf("- %s (%d)", truncate(item.Message, 120), item.Count))
	}
	if len(lines) == 0 {
		return []string{"- 없음"}
	}
	return lines
}

// truncate는 보고 메시지 길이를 줄이기 위해 긴 문자열을 잘라낸다.
func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return strings.TrimSpace(value[:max-3]) + "..."
}
