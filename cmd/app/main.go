package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"dsa/internal/config"
	"dsa/internal/job"
	"dsa/internal/logx"
	"dsa/internal/scheduler"
)

// main은 배치 프로그램의 진입점이다.
//
// 지원 명령:
// - 실행 없이 예시 설정 파일 생성
// - 작업 스케줄러 등록/삭제
// - keepalive / daily-report 직접 실행
func main() {
	command, runMode, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if command == "init-config" {
		exePath, err := os.Executable()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		configPath := filepath.Join(filepath.Dir(exePath), config.DefaultConfigName)
		if err := config.SaveExample(configPath); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("설정 파일 생성:", configPath)
		return
	}

	cfg, err := config.Load(runMode)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if command == "register-tasks" {
		exePath, err := os.Executable()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := scheduler.RegisterWindowsTasks(cfg, exePath); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("작업 스케줄러 등록 완료")
		return
	}

	if command == "unregister-tasks" {
		if err := scheduler.UnregisterWindowsTasks(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("작업 스케줄러 삭제 완료")
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	appLogger := logx.New(cfg.LogDir, logger)

	ctx := context.Background()
	if err := job.Run(ctx, cfg, appLogger); err != nil {
		logger.Error("job failed", "mode", cfg.RunMode, "error", err)
		os.Exit(1)
	}
}

func parseArgs(args []string) (command string, runMode string, err error) {
	if len(args) == 0 {
		return "run", "", nil
	}

	switch args[0] {
	case "run":
		if len(args) == 1 {
			return "run", "", nil
		}
		if args[1] != config.RunModeKeepalive && args[1] != config.RunModeDailyReport {
			return "", "", fmt.Errorf("지원하지 않는 실행 모드: %s", args[1])
		}
		return "run", args[1], nil
	case "init-config":
		return "init-config", "", nil
	case "register-tasks":
		return "register-tasks", "", nil
	case "unregister-tasks":
		return "unregister-tasks", "", nil
	default:
		return "", "", fmt.Errorf("지원하지 않는 명령: %s", args[0])
	}
}
