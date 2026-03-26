//go:build windows

package gui

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"dsa/internal/config"
	"dsa/internal/job"
	"dsa/internal/logx"
	"dsa/internal/scheduler"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type appWindow struct {
	mw              *walk.MainWindow
	ni              *walk.NotifyIcon
	statusLabel     *walk.Label
	databaseURL     *walk.LineEdit
	webhookURL      *walk.LineEdit
	dbLabel         *walk.LineEdit
	timezone        *walk.LineEdit
	logDir          *walk.LineEdit
	queryTimeout    *walk.NumberEdit
	httpTimeout     *walk.NumberEdit
	retryCount      *walk.NumberEdit
	retentionDays   *walk.NumberEdit
	keepaliveHours  *walk.NumberEdit
	dailyReportHour *walk.NumberEdit
}

// Run은 설정 창과 트레이 아이콘을 실행한다.
func Run() error {
	configPath, _, err := config.ResolvePaths()
	if err != nil {
		return err
	}

	cfg, err := config.Load("")
	if err != nil {
		cfg = defaultGUIConfig(configPath)
	} else if cfg.ConfigPath == "" {
		cfg.ConfigPath = configPath
	}

	aw := &appWindow{}
	if err := aw.create(cfg); err != nil {
		return err
	}
	defer func() {
		if aw.ni != nil {
			aw.ni.Dispose()
		}
	}()

	aw.refreshTaskState()
	aw.setStatus("대기 중")
	aw.mw.Show()
	aw.mw.Run()
	return nil
}

func defaultGUIConfig(configPath string) config.Config {
	logDir := "logs"
	if configPath != "" {
		logDir = filepath.Join(filepath.Dir(configPath), "logs")
	}
	cfg := config.Config{
		RunMode:               config.RunModeKeepalive,
		AppTimezone:           "Asia/Seoul",
		LogDir:                logDir,
		MaxRetryCount:         1,
		LogRetentionDays:      7,
		DBLabel:               "gabiadb",
		QueryTimeoutSeconds:   5,
		HTTPTimeoutSeconds:    5,
		KeepaliveIntervalHour: 3,
		DailyReportHour:       9,
		ConfigPath:            configPath,
	}
	cfg.QueryTimeout = time.Duration(cfg.QueryTimeoutSeconds) * time.Second
	cfg.HTTPTimeout = time.Duration(cfg.HTTPTimeoutSeconds) * time.Second
	return cfg
}

func (aw *appWindow) create(cfg config.Config) error {
	err := (MainWindow{
		AssignTo: &aw.mw,
		Title:    "DSA 설정",
		MinSize:  Size{Width: 720, Height: 580},
		Layout:   VBox{},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 2},
				Children: []Widget{
					Label{Text: "DATABASE_URL"},
					LineEdit{AssignTo: &aw.databaseURL, Text: cfg.DatabaseURL},

					Label{Text: "카카오워크 웹훅 URL"},
					LineEdit{AssignTo: &aw.webhookURL, Text: cfg.KakaoWorkWebhookURL},

					Label{Text: "DB 표시 이름"},
					LineEdit{AssignTo: &aw.dbLabel, Text: cfg.DBLabel},

					Label{Text: "타임존"},
					LineEdit{AssignTo: &aw.timezone, Text: cfg.AppTimezone},

					Label{Text: "로그 폴더"},
					LineEdit{AssignTo: &aw.logDir, Text: cfg.LogDir},

					Label{Text: "DB 타임아웃(초)"},
					NumberEdit{AssignTo: &aw.queryTimeout, Value: float64(cfg.QueryTimeoutSeconds), MinValue: 1, MaxValue: 600, Decimals: 0},

					Label{Text: "웹훅 타임아웃(초)"},
					NumberEdit{AssignTo: &aw.httpTimeout, Value: float64(cfg.HTTPTimeoutSeconds), MinValue: 1, MaxValue: 600, Decimals: 0},

					Label{Text: "재시도 횟수"},
					NumberEdit{AssignTo: &aw.retryCount, Value: float64(cfg.MaxRetryCount), MinValue: 0, MaxValue: 10, Decimals: 0},

					Label{Text: "로그 보관 일수"},
					NumberEdit{AssignTo: &aw.retentionDays, Value: float64(cfg.LogRetentionDays), MinValue: 1, MaxValue: 365, Decimals: 0},

					Label{Text: "연결유지 주기(시간)"},
					NumberEdit{AssignTo: &aw.keepaliveHours, Value: float64(cfg.KeepaliveIntervalHour), MinValue: 1, MaxValue: 24, Decimals: 0},

					Label{Text: "일일 보고 시각(0-23)"},
					NumberEdit{AssignTo: &aw.dailyReportHour, Value: float64(cfg.DailyReportHour), MinValue: 0, MaxValue: 23, Decimals: 0},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					PushButton{Text: "설정 저장", OnClicked: func() { aw.saveConfig() }},
					PushButton{Text: "작업 등록", OnClicked: func() { aw.registerTasks() }},
					PushButton{Text: "작업 중지", OnClicked: func() { aw.disableTasks() }},
					PushButton{Text: "작업 재개", OnClicked: func() { aw.enableTasks() }},
					PushButton{Text: "작업 삭제", OnClicked: func() { aw.unregisterTasks() }},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					PushButton{Text: "연결유지 실행", OnClicked: func() { aw.runNow(config.RunModeKeepalive) }},
					PushButton{Text: "일일 보고 실행", OnClicked: func() { aw.runNow(config.RunModeDailyReport) }},
					PushButton{Text: "트레이로 숨기기", OnClicked: func() { aw.hideToTray() }},
					PushButton{Text: "종료", OnClicked: func() { walk.App().Exit(0) }},
				},
			},
			Label{AssignTo: &aw.statusLabel, Text: "대기 중"},
		},
	}).Create()
	if err != nil {
		return err
	}

	aw.mw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		*canceled = true
		aw.hideToTray()
	})

	return aw.initTray()
}

func (aw *appWindow) initTray() error {
	icon, err := walk.NewIconFromSysDLL("shell32", 44)
	if err != nil {
		return err
	}
	if err := aw.mw.SetIcon(icon); err != nil {
		return err
	}

	ni, err := walk.NewNotifyIcon(aw.mw)
	if err != nil {
		return err
	}
	aw.ni = ni

	if err := aw.ni.SetIcon(icon); err != nil {
		return err
	}
	if err := aw.ni.SetToolTip("DSA"); err != nil {
		return err
	}

	openAction := walk.NewAction()
	openAction.SetText("설정 창 열기")
	openAction.Triggered().Attach(func() {
		aw.mw.Show()
		aw.mw.SetFocus()
	})
	_ = aw.ni.ContextMenu().Actions().Add(openAction)

	keepaliveAction := walk.NewAction()
	keepaliveAction.SetText("연결유지 실행")
	keepaliveAction.Triggered().Attach(func() { aw.runNow(config.RunModeKeepalive) })
	_ = aw.ni.ContextMenu().Actions().Add(keepaliveAction)

	reportAction := walk.NewAction()
	reportAction.SetText("일일 보고 실행")
	reportAction.Triggered().Attach(func() { aw.runNow(config.RunModeDailyReport) })
	_ = aw.ni.ContextMenu().Actions().Add(reportAction)

	disableAction := walk.NewAction()
	disableAction.SetText("작업 중지")
	disableAction.Triggered().Attach(func() { aw.disableTasks() })
	_ = aw.ni.ContextMenu().Actions().Add(disableAction)

	enableAction := walk.NewAction()
	enableAction.SetText("작업 재개")
	enableAction.Triggered().Attach(func() { aw.enableTasks() })
	_ = aw.ni.ContextMenu().Actions().Add(enableAction)

	exitAction := walk.NewAction()
	exitAction.SetText("종료")
	exitAction.Triggered().Attach(func() { walk.App().Exit(0) })
	_ = aw.ni.ContextMenu().Actions().Add(exitAction)

	aw.ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			aw.mw.Show()
			aw.mw.SetFocus()
		}
	})

	return aw.ni.SetVisible(true)
}

func (aw *appWindow) currentConfig() (config.Config, error) {
	configPath, _, err := config.ResolvePaths()
	if err != nil {
		return config.Config{}, err
	}

	cfg := config.Config{
		RunMode:               config.RunModeKeepalive,
		DatabaseURL:           aw.databaseURL.Text(),
		KakaoWorkWebhookURL:   aw.webhookURL.Text(),
		AppTimezone:           aw.timezone.Text(),
		LogDir:                aw.logDir.Text(),
		MaxRetryCount:         int(aw.retryCount.Value()),
		LogRetentionDays:      int(aw.retentionDays.Value()),
		DBLabel:               aw.dbLabel.Text(),
		QueryTimeoutSeconds:   int(aw.queryTimeout.Value()),
		HTTPTimeoutSeconds:    int(aw.httpTimeout.Value()),
		KeepaliveIntervalHour: int(aw.keepaliveHours.Value()),
		DailyReportHour:       int(aw.dailyReportHour.Value()),
		ConfigPath:            configPath,
	}
	cfg.QueryTimeout = time.Duration(cfg.QueryTimeoutSeconds) * time.Second
	cfg.HTTPTimeout = time.Duration(cfg.HTTPTimeoutSeconds) * time.Second

	if cfg.DatabaseURL == "" {
		return config.Config{}, fmt.Errorf("DATABASE_URL 값을 입력해 주세요")
	}
	if cfg.KakaoWorkWebhookURL == "" {
		return config.Config{}, fmt.Errorf("카카오워크 웹훅 URL 값을 입력해 주세요")
	}
	return cfg, nil
}

func (aw *appWindow) saveConfig() {
	cfg, err := aw.currentConfig()
	if err != nil {
		aw.showError(err)
		return
	}
	if err := config.Save(cfg.ConfigPath, cfg); err != nil {
		aw.showError(err)
		return
	}

	_, dotEnvPath, err := config.ResolvePaths()
	if err == nil {
		if backupPath, backupErr := config.BackupDotEnv(dotEnvPath); backupErr == nil && backupPath != "" {
			aw.setStatus("설정 저장 완료 / .env 백업: " + backupPath)
			return
		}
	}
	aw.setStatus("설정 저장 완료")
}

func (aw *appWindow) registerTasks() {
	cfg, err := aw.currentConfig()
	if err != nil {
		aw.showError(err)
		return
	}
	if err := config.Save(cfg.ConfigPath, cfg); err != nil {
		aw.showError(err)
		return
	}
	exePath, err := os.Executable()
	if err != nil {
		aw.showError(err)
		return
	}
	if err := scheduler.RegisterWindowsTasks(cfg, exePath); err != nil {
		aw.showError(err)
		return
	}
	aw.refreshTaskState()
	aw.setStatus("작업 스케줄러 등록 완료")
}

func (aw *appWindow) unregisterTasks() {
	if err := scheduler.UnregisterWindowsTasks(); err != nil {
		aw.showError(err)
		return
	}
	aw.refreshTaskState()
	aw.setStatus("작업 스케줄러 삭제 완료")
}

func (aw *appWindow) disableTasks() {
	if err := scheduler.DisableWindowsTasks(); err != nil {
		aw.showError(err)
		return
	}
	aw.refreshTaskState()
	aw.setStatus("작업 중지 완료")
}

func (aw *appWindow) enableTasks() {
	if err := scheduler.EnableWindowsTasks(); err != nil {
		aw.showError(err)
		return
	}
	aw.refreshTaskState()
	aw.setStatus("작업 재개 완료")
}

func (aw *appWindow) runNow(runMode string) {
	cfg, err := aw.currentConfig()
	if err != nil {
		aw.showError(err)
		return
	}
	cfg.RunMode = runMode

	go func() {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
		appLogger := logx.New(cfg.LogDir, logger)
		err := job.Run(context.Background(), cfg, appLogger)
		aw.mw.Synchronize(func() {
			if err != nil {
				aw.showError(err)
				return
			}
			aw.setStatus("즉시 실행 완료")
		})
	}()

	aw.setStatus("실행 중...")
}

func (aw *appWindow) hideToTray() {
	aw.mw.Hide()
	aw.setStatus("트레이로 숨김")
}

func (aw *appWindow) refreshTaskState() {
	state, err := scheduler.QueryWindowsTaskState()
	if err != nil {
		aw.setStatus("작업 상태 조회 실패")
		return
	}
	aw.setStatus(fmt.Sprintf(
		"연결유지: 등록=%t 활성=%t / 일일보고: 등록=%t 활성=%t",
		state.KeepaliveExists,
		state.KeepaliveEnabled,
		state.DailyReportExists,
		state.DailyReportEnabled,
	))
}

func (aw *appWindow) setStatus(text string) {
	if aw.statusLabel != nil {
		aw.statusLabel.SetText(text)
	}
}

func (aw *appWindow) showError(err error) {
	aw.setStatus("오류: " + err.Error())
	walk.MsgBox(aw.mw, "오류", err.Error(), walk.MsgBoxIconError)
}
