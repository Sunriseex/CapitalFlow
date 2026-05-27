package config

import (
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/sunriseex/capitalflow/pkg/errors"
)

type Config struct {
	AppEnv                    string
	TelegramToken             string
	TelegramUserID            int64
	AppVersion                string
	DataPath                  string
	DepositsDataPath          string
	DatabaseURL               string
	APIAuthToken              string
	JWTSecret                 string
	AccessTokenTTL            time.Duration
	RefreshTokenTTL           time.Duration
	PublicOrigin              string
	PublicOriginHost          string
	CookieSecure              bool
	CookieSameSite            string
	AllowDirectIPLogin        bool
	CORSAllowedOrigins        []string
	RateLimitRequests         int
	RateLimitWindow           time.Duration
	AuthRateLimitRequests     int
	AuthRateLimitWindow       time.Duration
	MutationRateLimitRequests int
	MutationRateLimitWindow   time.Duration
	TrustedProxies            []string
	LogLevel                  slog.Level
}

var AppConfig *Config

const MinAuthSecretLength = 32

func Init() error {
	envPaths := []string{"./configs/.env"}
	if envFile := strings.TrimSpace(os.Getenv("CAPITALFLOW_ENV_FILE")); envFile != "" {
		envPaths = append([]string{envFile}, envPaths...)
	}

	var loaded bool
	for _, envPath := range envPaths {
		if err := godotenv.Load(envPath); err == nil {
			loaded = true
			break
		}
	}

	if !loaded {
		slog.Debug("env file not found, using defaults")
	}

	dataPath, err := expandPath(getEnv("DATA_PATH", "~/.config/waybar/payments.json"))
	if err != nil {
		return errors.NewConfigurationError("ошибка расширения пути DATA_PATH", err)
	}

	depositsDataPath, err := expandPath(getEnv("DEPOSITS_DATA_PATH", "~/.config/waybar/deposits.json"))
	if err != nil {
		return errors.NewConfigurationError("ошибка расширения пути DEPOSITS_DATA_PATH", err)
	}

	logLevel := slog.LevelError
	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		switch envLogLevel {
		case "debug":
			logLevel = slog.LevelDebug
		case "info":
			logLevel = slog.LevelInfo
		case "warn":
			logLevel = slog.LevelWarn
		case "error":
			logLevel = slog.LevelError
		}
	}

	AppConfig = &Config{
		AppEnv:           getEnv("APP_ENV", "development"),
		TelegramToken:    getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramUserID:   getEnvInt64("TELEGRAM_USER_ID", 0),
		AppVersion:       getEnv("APP_VERSION", "0.1.0-dev"),
		DataPath:         dataPath,
		DepositsDataPath: depositsDataPath,
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://capitalflow:capitalflow@localhost:5432/capitalflow?sslmode=disable"),
		LogLevel:         logLevel,
		APIAuthToken:     getEnv("API_AUTH_TOKEN", ""),
		JWTSecret:        getEnv("JWT_SECRET", ""),
		AccessTokenTTL:   getEnvDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:  getEnvDuration("REFRESH_TOKEN_TTL", 30*24*time.Hour),
		PublicOrigin:     getEnv("PUBLIC_ORIGIN", ""),
		CookieSameSite:   getEnv("COOKIE_SAMESITE", "Strict"),
		CORSAllowedOrigins: getEnvList("CORS_ALLOWED_ORIGINS", []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
		}),
		RateLimitRequests:         getEnvInt("RATE_LIMIT_REQUESTS", 120),
		RateLimitWindow:           getEnvDuration("RATE_LIMIT_WINDOW", time.Minute),
		AuthRateLimitRequests:     getEnvInt("AUTH_RATE_LIMIT_REQUESTS", 5),
		AuthRateLimitWindow:       getEnvDuration("AUTH_RATE_LIMIT_WINDOW", time.Minute),
		MutationRateLimitRequests: getEnvInt("MUTATION_RATE_LIMIT_REQUESTS", 60),
		MutationRateLimitWindow:   getEnvDuration("MUTATION_RATE_LIMIT_WINDOW", time.Minute),
		TrustedProxies:            getEnvList("TRUSTED_PROXIES", nil),
	}
	AppConfig.AppEnv, err = normalizeAppEnv(AppConfig.AppEnv)
	if err != nil {
		return err
	}
	AppConfig.CookieSecure = getEnvBool("COOKIE_SECURE", true)
	AppConfig.AllowDirectIPLogin = getEnvBool("ALLOW_DIRECT_IP_LOGIN", !AppConfig.IsProduction())

	publicOrigin, publicOriginHost, err := parsePublicOrigin(AppConfig.PublicOrigin)
	if err != nil {
		return err
	}
	AppConfig.PublicOrigin = publicOrigin
	AppConfig.PublicOriginHost = publicOriginHost
	AppConfig.CookieSameSite, err = normalizeCookieSameSite(AppConfig.CookieSameSite)
	if err != nil {
		return err
	}
	if err := AppConfig.ValidateCookiePolicy(); err != nil {
		return err
	}
	if err := AppConfig.ValidateSecurity(); err != nil {
		return err
	}

	initLogger(logLevel)

	slog.Debug("Конфигурация инициализирована",
		"data_path", dataPath,
		"deposit_path", depositsDataPath,
		"log_level", logLevel)

	return nil
}

func ValidateAuthSecret(name, value string) error {
	if len(strings.TrimSpace(value)) < MinAuthSecretLength {
		return errors.NewConfigurationError(name+" must be at least 32 characters", nil)
	}
	return nil
}

func (c *Config) IsProduction() bool {
	return strings.EqualFold(strings.TrimSpace(c.AppEnv), "production")
}

func (c *Config) ValidateSecurity() error {
	if c == nil || !c.IsProduction() {
		return nil
	}
	if c.PublicOrigin == "" {
		return errors.NewConfigurationError("PUBLIC_ORIGIN is required in production", nil)
	}
	if err := ValidateAuthSecret("JWT_SECRET", c.JWTSecret); err != nil {
		return err
	}
	if isPlaceholderSecret(c.JWTSecret) {
		return errors.NewConfigurationError("JWT_SECRET must not use a placeholder value in production", nil)
	}
	return nil
}

func (c *Config) ValidateCookiePolicy() error {
	if c == nil {
		return nil
	}
	if c.CookieSameSite == "None" && !c.CookieSecure {
		return errors.NewConfigurationError("COOKIE_SAMESITE=None requires COOKIE_SECURE=true", nil)
	}
	return nil
}

func initLogger(level slog.Level) {
	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler

	if level == slog.LevelDebug {
		handler = slog.NewTextHandler(os.Stderr, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}
	slog.SetDefault(slog.New(handler))
}

func expandPath(path string) (string, error) {
	if path == "" {
		return "", errors.NewConfigurationError("путь не может быть пустым", nil)
	}

	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", errors.NewConfigurationError("не удалось получить домашнюю директорию", err)
		}

		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", errors.NewConfigurationError("ошибка получения абсолютного пути", err)
	}
	return absPath, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	if parsed, err := strconv.ParseBool(value); err == nil {
		return parsed
	}
	return defaultValue
}

func getEnvList(key string, defaultValue []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}

	items := make([]string, 0)
	for part := range strings.SplitSeq(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			items = append(items, part)
		}
	}
	if len(items) == 0 {
		return defaultValue
	}
	return items
}

func normalizeAppEnv(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "", "development":
		return "development", nil
	case "production":
		return "production", nil
	default:
		return "", errors.NewConfigurationError("APP_ENV must be development or production", nil)
	}
}

func parsePublicOrigin(value string) (origin, host string, err error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil ||
		parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", "", errors.NewConfigurationError("PUBLIC_ORIGIN must be a full origin without path, query, or fragment", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", "", errors.NewConfigurationError("PUBLIC_ORIGIN scheme must be http or https", nil)
	}
	origin = canonicalOrigin(parsed.Scheme, parsed.Host)
	originURL, err := url.Parse(origin)
	if err != nil {
		return "", "", errors.NewConfigurationError("PUBLIC_ORIGIN must be a valid origin", err)
	}
	return origin, originURL.Host, nil
}

func normalizeCookieSameSite(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "strict":
		return "Strict", nil
	case "lax":
		return "Lax", nil
	case "none":
		return "None", nil
	default:
		return "", errors.NewConfigurationError("COOKIE_SAMESITE must be Strict, Lax, or None", nil)
	}
}

func isPlaceholderSecret(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return true
	}
	placeholders := []string{
		"change-me",
		"change-me-to-at-least-32-random-bytes",
		"change-me-to-a-long-random-secret",
		"your-jwt-secret",
		"secret",
	}
	for _, placeholder := range placeholders {
		if normalized == placeholder {
			return true
		}
	}
	return false
}

func canonicalOrigin(scheme, host string) string {
	scheme = strings.ToLower(strings.TrimSpace(scheme))
	host = strings.ToLower(strings.TrimSpace(host))
	switch {
	case scheme == "https" && strings.HasSuffix(host, ":443"):
		host = strings.TrimSuffix(host, ":443")
	case scheme == "http" && strings.HasSuffix(host, ":80"):
		host = strings.TrimSuffix(host, ":80")
	}
	return scheme + "://" + host
}
