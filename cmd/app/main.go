package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"dsa/internal/config"
	"dsa/internal/gui"
	"dsa/internal/job"
	"dsa/internal/logx"
	"dsa/internal/scheduler"
	"dsa/internal/setupwizard"
)

// main은 배치 프로그램의 진입점이다.
//
// 지원 명령:
// - gui: 설정 창과 트레이 앱 실행
// - setup: 설정 파일 생성/수정 + 작업 스케줄러 등록
// - init-config: 예시 설정 파일만 생성
// - register-tasks: 현재 설정 기준으로 작업 스케줄러 등록
// - unregister-tasks: 작업 스케줄러 삭제
// - run keepalive|daily-report: 직접 실행
func main() {
	command, runMode, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if command == "init-config" {
		configPath, _, err := config.ResolvePaths()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := config.SaveExample(configPath); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("설정 파일 생성:", configPath)
		return
	}

	if command == "gui" {
		if err := gui.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if command == "setup" {
		exePath, err := os.Executable()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		configPath, dotEnvPath, err := config.ResolvePaths()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		cfg, registerTasks, err := setupwizard.Run(configPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := config.Save(configPath, cfg); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("설정 파일 저장 완료:", configPath)

		backupPath, err := config.BackupDotEnv(dotEnvPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if backupPath != "" {
			fmt.Println(".env 백업 완료:", backupPath)
		}

		if registerTasks {
			if err := scheduler.RegisterWindowsTasks(cfg, exePath); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println("작업 스케줄러 등록 완료")
		}
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
		return "gui", "", nil
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
	case "gui":
		return "gui", "", nil
	case "setup":
		return "setup", "", nil
	case "register-tasks":
		return "register-tasks", "", nil
	case "unregister-tasks":
		return "unregister-tasks", "", nil
	default:
		return "", "", fmt.Errorf("지원하지 않는 명령: %s", args[0])
	}
}
