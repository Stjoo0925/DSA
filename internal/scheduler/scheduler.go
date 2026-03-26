package scheduler

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"

	"dsa/internal/config"
)

const (
	keepaliveTaskName = "DSA Keepalive"
	reportTaskName    = "DSA Daily Report"
)

// RegisterWindowsTasks는 현재 실행 파일을 기준으로 윈도우 작업 스케줄러를 등록한다.
func RegisterWindowsTasks(cfg config.Config, exePath string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("작업 스케줄러 자동 등록은 Windows에서만 지원합니다")
	}

	workDir := filepath.Dir(exePath)

	keepaliveCmd := fmt.Sprintf("cmd /c \"cd /d \"%s\" && \"%s\" run keepalive\"", workDir, exePath)
	reportCmd := fmt.Sprintf("cmd /c \"cd /d \"%s\" && \"%s\" run daily-report\"", workDir, exePath)

	if err := runSCHTASKS(
		"/Create",
		"/F",
		"/TN", keepaliveTaskName,
		"/SC", "HOURLY",
		"/MO", fmt.Sprintf("%d", cfg.KeepaliveIntervalHour),
		"/TR", keepaliveCmd,
		"/RL", "HIGHEST",
		"/RU", "SYSTEM",
		"/ST", "00:00",
	); err != nil {
		return fmt.Errorf("keepalive 작업 등록 실패: %w", err)
	}

	if err := runSCHTASKS(
		"/Create",
		"/F",
		"/TN", reportTaskName,
		"/SC", "DAILY",
		"/TR", reportCmd,
		"/RL", "HIGHEST",
		"/RU", "SYSTEM",
		"/ST", fmt.Sprintf("%02d:00", cfg.DailyReportHour),
	); err != nil {
		return fmt.Errorf("daily-report 작업 등록 실패: %w", err)
	}

	return nil
}

// UnregisterWindowsTasks는 DSA가 만든 작업 스케줄러 항목을 삭제한다.
func UnregisterWindowsTasks() error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("작업 스케줄러 자동 삭제는 Windows에서만 지원합니다")
	}

	_ = runSCHTASKS("/Delete", "/F", "/TN", keepaliveTaskName)
	_ = runSCHTASKS("/Delete", "/F", "/TN", reportTaskName)
	return nil
}

func runSCHTASKS(args ...string) error {
	cmd := exec.Command("schtasks", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	return nil
}
