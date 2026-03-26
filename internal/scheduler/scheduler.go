package scheduler

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"dsa/internal/config"
)

const (
	keepaliveTaskName = "DSA Keepalive"
	reportTaskName    = "DSA Daily Report"
)

type TaskState struct {
	KeepaliveExists    bool
	DailyReportExists  bool
	KeepaliveEnabled   bool
	DailyReportEnabled bool
}

// RegisterWindowsTasks는 현재 실행 파일을 기준으로 윈도우 작업 스케줄러를 등록한다.
func RegisterWindowsTasks(cfg config.Config, exePath string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("작업 스케줄러 자동 등록은 Windows에서만 지원합니다")
	}

	workDir := filepath.Dir(exePath)

	// cmd /c 내부의 경로 공백 처리: 중첩 따옴표를 \"로 이스케이프한다.
	keepaliveCmd := fmt.Sprintf(`cmd /c "cd /d \"%s\" && \"%s\" run keepalive"`, workDir, exePath)
	reportCmd := fmt.Sprintf(`cmd /c "cd /d \"%s\" && \"%s\" run daily-report"`, workDir, exePath)

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

	var errs []error
	if err := runSCHTASKS("/Delete", "/F", "/TN", keepaliveTaskName); err != nil {
		errs = append(errs, fmt.Errorf("keepalive 작업 삭제 실패: %w", err))
	}
	if err := runSCHTASKS("/Delete", "/F", "/TN", reportTaskName); err != nil {
		errs = append(errs, fmt.Errorf("daily-report 작업 삭제 실패: %w", err))
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func DisableWindowsTasks() error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("작업 스케줄러 비활성화는 Windows에서만 지원합니다")
	}

	var errs []error
	if err := runSCHTASKS("/Change", "/TN", keepaliveTaskName, "/Disable"); err != nil {
		errs = append(errs, fmt.Errorf("keepalive 작업 비활성화 실패: %w", err))
	}
	if err := runSCHTASKS("/Change", "/TN", reportTaskName, "/Disable"); err != nil {
		errs = append(errs, fmt.Errorf("daily-report 작업 비활성화 실패: %w", err))
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func EnableWindowsTasks() error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("작업 스케줄러 활성화는 Windows에서만 지원합니다")
	}

	var errs []error
	if err := runSCHTASKS("/Change", "/TN", keepaliveTaskName, "/Enable"); err != nil {
		errs = append(errs, fmt.Errorf("keepalive 작업 활성화 실패: %w", err))
	}
	if err := runSCHTASKS("/Change", "/TN", reportTaskName, "/Enable"); err != nil {
		errs = append(errs, fmt.Errorf("daily-report 작업 활성화 실패: %w", err))
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func QueryWindowsTaskState() (TaskState, error) {
	if runtime.GOOS != "windows" {
		return TaskState{}, fmt.Errorf("작업 스케줄러 상태 조회는 Windows에서만 지원합니다")
	}

	keepExists, keepEnabled, err := querySingleTaskState(keepaliveTaskName)
	if err != nil {
		return TaskState{}, err
	}
	reportExists, reportEnabled, err := querySingleTaskState(reportTaskName)
	if err != nil {
		return TaskState{}, err
	}

	return TaskState{
		KeepaliveExists:    keepExists,
		DailyReportExists:  reportExists,
		KeepaliveEnabled:   keepEnabled,
		DailyReportEnabled: reportEnabled,
	}, nil
}

func querySingleTaskState(taskName string) (exists bool, enabled bool, err error) {
	cmd := exec.Command("schtasks", "/Query", "/TN", taskName, "/FO", "LIST", "/V")
	output, err := cmd.CombinedOutput()
	if err != nil {
		text := string(output)
		if strings.Contains(text, "ERROR: The system cannot find the file specified.") ||
			strings.Contains(text, "오류: 지정된 파일을 찾을 수 없습니다.") {
			return false, false, nil
		}
		return false, false, fmt.Errorf("작업 상태 조회 실패(%s): %w: %s", taskName, err, text)
	}

	text := string(output)
	upper := strings.ToUpper(text)
	enabled = strings.Contains(upper, "SCHEDULED TASK STATE: ENABLED") ||
		strings.Contains(text, "예약된 작업 상태: 사용")

	return true, enabled, nil
}

func runSCHTASKS(args ...string) error {
	cmd := exec.Command("schtasks", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	return nil
}
