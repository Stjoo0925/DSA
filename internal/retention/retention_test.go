package retention

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func makeFile(t *testing.T, dir, name string) {
	t.Helper()
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	f.Close()
}

func TestCleanup_MissingDirReturnsNil(t *testing.T) {
	err := Cleanup("/nonexistent/path/does/not/exist", 7, time.Now())
	if err != nil {
		t.Errorf("Cleanup() missing dir = %v, want nil", err)
	}
}

func TestCleanup_DeletesOldFiles(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 3, 26, 0, 0, 0, 0, time.UTC)

	// 8일 전 파일 → 삭제 대상 (retentionDays=7)
	makeFile(t, dir, "2026-03-18.jsonl")
	// 6일 전 파일 → 보관
	makeFile(t, dir, "2026-03-20.jsonl")

	if err := Cleanup(dir, 7, now); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "2026-03-18.jsonl")); !os.IsNotExist(err) {
		t.Error("2026-03-18.jsonl should have been deleted")
	}
	if _, err := os.Stat(filepath.Join(dir, "2026-03-20.jsonl")); err != nil {
		t.Errorf("2026-03-20.jsonl should still exist: %v", err)
	}
}

func TestCleanup_KeepsCutoffBoundaryFile(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 3, 26, 0, 0, 0, 0, time.UTC)

	// 정확히 7일 전 → cutoff와 같은 날 → Before(cutoff) == false → 보관
	makeFile(t, dir, "2026-03-19.jsonl")

	if err := Cleanup(dir, 7, now); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "2026-03-19.jsonl")); err != nil {
		t.Errorf("cutoff-day file should be kept: %v", err)
	}
}

func TestCleanup_IgnoresNonJsonlFiles(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 3, 26, 0, 0, 0, 0, time.UTC)

	makeFile(t, dir, "2020-01-01.log")
	makeFile(t, dir, "2020-01-01.txt")

	if err := Cleanup(dir, 7, now); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "2020-01-01.log")); err != nil {
		t.Error(".log file should not be touched")
	}
	if _, err := os.Stat(filepath.Join(dir, "2020-01-01.txt")); err != nil {
		t.Error(".txt file should not be touched")
	}
}

func TestCleanup_IgnoresMalformedJsonlNames(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 3, 26, 0, 0, 0, 0, time.UTC)

	makeFile(t, dir, "not-a-date.jsonl")
	makeFile(t, dir, "2026.jsonl")

	if err := Cleanup(dir, 7, now); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "not-a-date.jsonl")); err != nil {
		t.Error("malformed name jsonl should not be deleted")
	}
}

func TestCleanup_IgnoresSubdirectories(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 3, 26, 0, 0, 0, 0, time.UTC)

	subDir := filepath.Join(dir, "2020-01-01.jsonl")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := Cleanup(dir, 7, now); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}

	if _, err := os.Stat(subDir); err != nil {
		t.Error("directory should not be deleted even if name matches pattern")
	}
}

func TestCleanup_EmptyDirReturnsNil(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	if err := Cleanup(dir, 7, now); err != nil {
		t.Errorf("Cleanup() empty dir = %v, want nil", err)
	}
}
