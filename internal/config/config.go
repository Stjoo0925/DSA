package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const (
	// RunModeKeepalive는 DB 유휴 방지 작업을 의미한다.
	RunModeKeepalive = "keepalive"
	// RunModeDailyReport는 전일 집계 보고 작업을 의미한다.
	RunModeDailyReport = "daily-report"
	DefaultConfigName  = "dsa.config.json"
)

// Config는 프로그램 실행에 필요한 모든 설정값을 담는다.
//
// 우선순위는 아래와 같다.
// 1. 명령행 인자에서 넘긴 실행 모드
// 2. 환경변수
// 3. 설정 파일
// 4. 코드 기본값
type Config struct {
	RunMode               string        `json:"run_mode"`
	DatabaseURL           string        `json:"database_url"`
	KakaoWorkWebhookURL   string        `json:"kakaowork_webhook_url"`
	AppTimezone           string        `json:"app_timezone"`
	LogDir                string        `json:"log_dir"`
	QueryTimeout          time.Duration `json:"-"`
	HTTPTimeout           time.Duration `json:"-"`
	MaxRetryCount         int           `json:"max_retry_count"`
	LogRetentionDays      int           `json:"log_retention_days"`
	DBLabel               string        `json:"db_label"`
	QueryTimeoutSeconds   int           `json:"query_timeout_seconds"`
	HTTPTimeoutSeconds    int           `json:"http_timeout_seconds"`
	KeepaliveIntervalHour int           `json:"keepalive_interval_hour"`
	DailyReportHour       int           `json:"daily_report_hour"`
	ConfigPath            string        `json:"-"`
}

// Load는 설정 파일과 환경변수를 읽어서 최종 설정값을 만든다.
func Load(runModeOverride string) (Config, error) {
	exePath, err := os.Executable()
	if err != nil {
		return Config{}, fmt.Errorf("resolve executable path: %w", err)
	}
	exeDir := filepath.Dir(exePath)
	configPath := filepath.Join(exeDir, DefaultConfigName)

	cfg := defaultConfig(exeDir)
	cfg.ConfigPath = configPath

	if fileCfg, err := loadFromFile(configPath); err == nil {
		merge(&cfg, fileCfg)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Config{}, err
	}

	mergeEnv(&cfg)

	if runModeOverride != "" {
		cfg.RunMode = runModeOverride
	}

	cfg.QueryTimeout = time.Duration(cfg.QueryTimeoutSeconds) * time.Second
	cfg.HTTPTimeout = time.Duration(cfg.HTTPTimeoutSeconds) * time.Second

	if cfg.RunMode != RunModeKeepalive && cfg.RunMode != RunModeDailyReport {
		return Config{}, fmt.Errorf("invalid RUN_MODE: %s", cfg.RunMode)
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL 또는 설정 파일의 database_url 값이 필요합니다")
	}
	if cfg.KakaoWorkWebhookURL == "" {
		return Config{}, errors.New("KAKAOWORK_WEBHOOK_URL 또는 설정 파일의 kakaowork_webhook_url 값이 필요합니다")
	}
	if cfg.MaxRetryCount < 0 {
		return Config{}, errors.New("MAX_RETRY_COUNT must be >= 0")
	}
	if cfg.LogRetentionDays < 1 {
		return Config{}, errors.New("LOG_RETENTION_DAYS must be >= 1")
	}
	if cfg.KeepaliveIntervalHour < 1 {
		return Config{}, errors.New("keepalive_interval_hour must be >= 1")
	}
	if cfg.DailyReportHour < 0 || cfg.DailyReportHour > 23 {
		return Config{}, errors.New("daily_report_hour must be between 0 and 23")
	}
	if _, err := time.LoadLocation(cfg.AppTimezone); err != nil {
		return Config{}, fmt.Errorf("invalid APP_TIMEZONE: %w", err)
	}

	return cfg, nil
}

// SaveExample은 실행 파일 옆에 기본 설정 파일 예시를 저장한다.
func SaveExample(path string) error {
	example := defaultConfig(filepath.Dir(path))
	example.DatabaseURL = "postgresql://username:password@host:5432/dbname?sslmode=require"
	example.KakaoWorkWebhookURL = "https://example.com/webhook"

	payload, err := json.MarshalIndent(example, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config example: %w", err)
	}

	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("write config example: %w", err)
	}
	return nil
}

func defaultConfig(baseDir string) Config {
	return Config{
		RunMode:               RunModeKeepalive,
		AppTimezone:           "Asia/Seoul",
		LogDir:                filepath.Join(baseDir, "logs"),
		MaxRetryCount:         1,
		LogRetentionDays:      7,
		DBLabel:               "gabiadb",
		QueryTimeoutSeconds:   5,
		HTTPTimeoutSeconds:    5,
		KeepaliveIntervalHour: 3,
		DailyReportHour:       9,
	}
}

func loadFromFile(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, os.ErrNotExist
		}
		return Config{}, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config file: %w", err)
	}
	return cfg, nil
}

func merge(dst *Config, src Config) {
	if src.RunMode != "" {
		dst.RunMode = src.RunMode
	}
	if src.DatabaseURL != "" {
		dst.DatabaseURL = src.DatabaseURL
	}
	if src.KakaoWorkWebhookURL != "" {
		dst.KakaoWorkWebhookURL = src.KakaoWorkWebhookURL
	}
	if src.AppTimezone != "" {
		dst.AppTimezone = src.AppTimezone
	}
	if src.LogDir != "" {
		dst.LogDir = src.LogDir
	}
	if src.MaxRetryCount != 0 {
		dst.MaxRetryCount = src.MaxRetryCount
	}
	if src.LogRetentionDays != 0 {
		dst.LogRetentionDays = src.LogRetentionDays
	}
	if src.DBLabel != "" {
		dst.DBLabel = src.DBLabel
	}
	if src.QueryTimeoutSeconds != 0 {
		dst.QueryTimeoutSeconds = src.QueryTimeoutSeconds
	}
	if src.HTTPTimeoutSeconds != 0 {
		dst.HTTPTimeoutSeconds = src.HTTPTimeoutSeconds
	}
	if src.KeepaliveIntervalHour != 0 {
		dst.KeepaliveIntervalHour = src.KeepaliveIntervalHour
	}
	if src.DailyReportHour != 0 {
		dst.DailyReportHour = src.DailyReportHour
	}
}

func mergeEnv(cfg *Config) {
	cfg.RunMode = getEnv("RUN_MODE", cfg.RunMode)
	cfg.DatabaseURL = getEnv("DATABASE_URL", cfg.DatabaseURL)
	cfg.KakaoWorkWebhookURL = getEnv("KAKAOWORK_WEBHOOK_URL", cfg.KakaoWorkWebhookURL)
	cfg.AppTimezone = getEnv("APP_TIMEZONE", cfg.AppTimezone)
	cfg.LogDir = getEnv("LOG_DIR", cfg.LogDir)
	cfg.DBLabel = getEnv("DB_LABEL", cfg.DBLabel)
	cfg.QueryTimeoutSeconds = getEnvInt("QUERY_TIMEOUT_SECONDS", cfg.QueryTimeoutSeconds)
	cfg.HTTPTimeoutSeconds = getEnvInt("HTTP_TIMEOUT_SECONDS", cfg.HTTPTimeoutSeconds)
	cfg.MaxRetryCount = getEnvInt("MAX_RETRY_COUNT", cfg.MaxRetryCount)
	cfg.LogRetentionDays = getEnvInt("LOG_RETENTION_DAYS", cfg.LogRetentionDays)
	cfg.KeepaliveIntervalHour = getEnvInt("KEEPALIVE_INTERVAL_HOUR", cfg.KeepaliveIntervalHour)
	cfg.DailyReportHour = getEnvInt("DAILY_REPORT_HOUR", cfg.DailyReportHour)
}

// getEnv는 환경변수가 있으면 그 값을 반환하고, 없으면 기본값을 반환한다.
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getEnvInt는 정수형 환경변수를 읽는다.
func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
