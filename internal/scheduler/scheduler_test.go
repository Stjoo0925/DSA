package scheduler

import (
	"runtime"
	"strings"
	"testing"

	"dsa/internal/config"
)

func TestRegisterWindowsTasks_NonWindowsReturnsError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("non-Windows 동작 검증 — Windows에서는 스킵")
	}

	cfg := config.Config{KeepaliveIntervalHour: 3, DailyReportHour: 9}
	err := RegisterWindowsTasks(cfg, "/usr/local/bin/dsa")
	if err == nil {
		t.Fatal("RegisterWindowsTasks() on non-Windows should return error")
	}
	if !strings.Contains(err.Error(), "Windows") {
		t.Errorf("error should mention Windows, got: %v", err)
	}
}

func TestUnregisterWindowsTasks_NonWindowsReturnsError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("non-Windows 동작 검증 — Windows에서는 스킵")
	}

	err := UnregisterWindowsTasks()
	if err == nil {
		t.Fatal("UnregisterWindowsTasks() on non-Windows should return error")
	}
	if !strings.Contains(err.Error(), "Windows") {
		t.Errorf("error should mention Windows, got: %v", err)
	}
}

func TestTaskNames_NotEmpty(t *testing.T) {
	if keepaliveTaskName == "" {
		t.Error("keepaliveTaskName must not be empty")
	}
	if reportTaskName == "" {
		t.Error("reportTaskName must not be empty")
	}
}

func TestTaskNames_Distinct(t *testing.T) {
	if keepaliveTaskName == reportTaskName {
		t.Errorf("task names must be different, both are %q", keepaliveTaskName)
	}
}
