package db

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestDefaultQueries_ReturnsThreeQueries(t *testing.T) {
	queries := DefaultQueries()
	if len(queries) != 3 {
		t.Errorf("DefaultQueries() len = %d, want 3", len(queries))
	}
}

func TestDefaultQueries_NamesAreCorrect(t *testing.T) {
	queries := DefaultQueries()
	want := []string{"simple_ping", "table_limit_probe", "full_table_probe"}
	for i, q := range queries {
		if q.Name != want[i] {
			t.Errorf("queries[%d].Name = %q, want %q", i, q.Name, want[i])
		}
	}
}

func TestDefaultQueries_SQLNotEmpty(t *testing.T) {
	for _, q := range DefaultQueries() {
		if q.SQL == "" {
			t.Errorf("query %q has empty SQL", q.Name)
		}
	}
}

func TestRunQuery_InvalidURLReturnsFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result := RunQuery(ctx, "postgres://invalid-host-that-does-not-exist:5432/db", QueryDefinition{
		Name: "test",
		SQL:  "SELECT 1",
	})

	if result.Success {
		t.Error("RunQuery() with invalid URL should return Success=false")
	}
	if result.Error == "" {
		t.Error("RunQuery() with invalid URL should set Error field")
	}
	if result.Name != "test" {
		t.Errorf("RunQuery() Name = %q, want %q", result.Name, "test")
	}
}

func TestRunQuery_CancelledContextReturnsFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 즉시 취소

	result := RunQuery(ctx, "postgres://localhost:5432/db", QueryDefinition{
		Name: "test",
		SQL:  "SELECT 1",
	})

	if result.Success {
		t.Error("RunQuery() with cancelled context should return Success=false")
	}
}

func TestRunQuery_DurationMSRecordedOnFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result := RunQuery(ctx, "postgres://invalid:5432/db", QueryDefinition{
		Name: "test",
		SQL:  "SELECT 1",
	})

	// 실패해도 duration은 기록된다
	if result.DurationMS < 0 {
		t.Errorf("DurationMS should be >= 0, got %d", result.DurationMS)
	}
}

// --- 통합 테스트 (DATABASE_URL 필요) ---

func TestRunQuery_Integration(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set — skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := RunQuery(ctx, url, QueryDefinition{
		Name: "simple_ping",
		SQL:  "SELECT 1;",
	})

	if !result.Success {
		t.Errorf("RunQuery() integration failed: %s", result.Error)
	}
	if result.DurationMS < 0 {
		t.Errorf("DurationMS = %d, want >= 0", result.DurationMS)
	}
}
