package setupwizard

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"dsa/internal/config"
)

// Run은 콘솔 기반 설정 마법사를 실행하고 결과 설정값을 반환한다.
func Run(configPath string) (config.Config, bool, error) {
	baseDir := filepath.Dir(configPath)
	cfg, err := config.Load("")
	if err != nil {
		cfg = config.Config{
			RunMode:               config.RunModeKeepalive,
			AppTimezone:           "Asia/Seoul",
			LogDir:                filepath.Join(baseDir, "logs"),
			MaxRetryCount:         1,
			LogRetentionDays:      7,
			DBLabel:               "gabiadb",
			QueryTimeoutSeconds:   5,
			HTTPTimeoutSeconds:    5,
			KeepaliveIntervalHour: 3,
			DailyReportHour:       9,
			ConfigPath:            configPath,
		}
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("DSA 설정 마법사")
	fmt.Println("빈 값으로 엔터를 누르면 현재 값 또는 기본값을 유지합니다.")
	fmt.Println(".env 파일이 있으면 그 값이 먼저 반영됩니다.")
	fmt.Println("설정 저장 후 기존 .env 파일은 백업 파일로 이름이 바뀝니다.")
	fmt.Println("")

	cfg.DatabaseURL = askString(reader, "DATABASE_URL", cfg.DatabaseURL)
	cfg.KakaoWorkWebhookURL = askString(reader, "카카오워크 웹훅 URL", cfg.KakaoWorkWebhookURL)
	cfg.DBLabel = askString(reader, "DB 표시 이름", cfg.DBLabel)
	cfg.AppTimezone = askString(reader, "타임존", cfg.AppTimezone)
	cfg.LogDir = askString(reader, "로그 폴더", cfg.LogDir)
	cfg.QueryTimeoutSeconds = askInt(reader, "DB 타임아웃(초)", cfg.QueryTimeoutSeconds)
	cfg.HTTPTimeoutSeconds = askInt(reader, "웹훅 타임아웃(초)", cfg.HTTPTimeoutSeconds)
	cfg.MaxRetryCount = askInt(reader, "재시도 횟수", cfg.MaxRetryCount)
	cfg.LogRetentionDays = askInt(reader, "로그 보관 일수", cfg.LogRetentionDays)
	cfg.KeepaliveIntervalHour = askInt(reader, "keepalive 실행 간격(시간)", cfg.KeepaliveIntervalHour)
	cfg.DailyReportHour = askInt(reader, "일일 보고 실행 시각(0-23)", cfg.DailyReportHour)

	registerTasks := askYesNo(reader, "작업 스케줄러도 같이 등록할까요? (y/n)", true)

	cfg.QueryTimeout = 0
	cfg.HTTPTimeout = 0
	cfg.ConfigPath = configPath

	return cfg, registerTasks, nil
}

func askString(reader *bufio.Reader, label, current string) string {
	fmt.Printf("%s [%s]: ", label, current)
	value, _ := reader.ReadString('\n')
	value = strings.TrimSpace(value)
	if value == "" {
		return current
	}
	return value
}

func askInt(reader *bufio.Reader, label string, current int) int {
	for {
		fmt.Printf("%s [%d]: ", label, current)
		value, _ := reader.ReadString('\n')
		value = strings.TrimSpace(value)
		if value == "" {
			return current
		}

		n, err := strconv.Atoi(value)
		if err == nil {
			return n
		}
		fmt.Println("숫자만 입력해 주세요.")
	}
}

func askYesNo(reader *bufio.Reader, label string, current bool) bool {
	defaultText := "y"
	if !current {
		defaultText = "n"
	}

	for {
		fmt.Printf("%s [%s]: ", label, defaultText)
		value, _ := reader.ReadString('\n')
		value = strings.TrimSpace(strings.ToLower(value))
		if value == "" {
			return current
		}
		if value == "y" || value == "yes" {
			return true
		}
		if value == "n" || value == "no" {
			return false
		}
		fmt.Println("y 또는 n으로 입력해 주세요.")
	}
}
