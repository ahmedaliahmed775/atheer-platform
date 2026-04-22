package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Security SecurityConfig
	Limits   LimitsConfig
}

type ServerConfig struct {
	Port            int           `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	TLSCertPath     string        `mapstructure:"tls_cert_path"`
	TLSKeyPath      string        `mapstructure:"tls_key_path"`
}

type DatabaseConfig struct {
	URL             string        `mapstructure:"url"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

type RedisConfig struct {
	URL      string `mapstructure:"url"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type SecurityConfig struct {
	KMSKeyARN              string `mapstructure:"kms_key_arn"`
	PlayIntegrityProjectID string `mapstructure:"play_integrity_project_id"`
	TxTimeWindowMinutes    int    `mapstructure:"tx_time_window_minutes"`
	IdempotencyTTLSeconds  int    `mapstructure:"idempotency_ttl_seconds"`
}

type LimitsConfig struct {
	RateLimitPerDevice int `mapstructure:"rate_limit_per_device"`
	RateLimitPerWallet int `mapstructure:"rate_limit_per_wallet"`
	RateLimitPerIP     int `mapstructure:"rate_limit_per_ip"`
}

// Load reads configuration from environment variables and .env file
func Load() (*Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", 30*time.Second)
	v.SetDefault("server.write_timeout", 30*time.Second)
	v.SetDefault("server.shutdown_timeout", 15*time.Second)

	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", 5*time.Minute)

	v.SetDefault("redis.db", 0)

	v.SetDefault("security.tx_time_window_minutes", 5)
	v.SetDefault("security.idempotency_ttl_seconds", 600)

	v.SetDefault("limits.rate_limit_per_device", 10)
	v.SetDefault("limits.rate_limit_per_wallet", 100)
	v.SetDefault("limits.rate_limit_per_ip", 50)

	// Environment variables
	v.SetEnvPrefix("ATHEER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Map specific env vars
	_ = v.BindEnv("database.url", "DATABASE_URL")
	_ = v.BindEnv("redis.url", "REDIS_URL")
	_ = v.BindEnv("redis.password", "REDIS_PASSWORD")
	_ = v.BindEnv("security.kms_key_arn", "KMS_KEY_ARN")
	_ = v.BindEnv("security.play_integrity_project_id", "PLAY_INTEGRITY_PROJECT_ID")
	_ = v.BindEnv("server.port", "PORT")
	_ = v.BindEnv("server.tls_cert_path", "TLS_CERT_PATH")
	_ = v.BindEnv("server.tls_key_path", "TLS_KEY_PATH")

	// Try .env file
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			slog.Warn("Error reading config file", "error", err)
		}
	}

	cfg := &Config{}

	cfg.Server = ServerConfig{
		Port:            v.GetInt("server.port"),
		ReadTimeout:     v.GetDuration("server.read_timeout"),
		WriteTimeout:    v.GetDuration("server.write_timeout"),
		ShutdownTimeout: v.GetDuration("server.shutdown_timeout"),
		TLSCertPath:     v.GetString("server.tls_cert_path"),
		TLSKeyPath:      v.GetString("server.tls_key_path"),
	}

	cfg.Database = DatabaseConfig{
		URL:             v.GetString("database.url"),
		MaxOpenConns:    v.GetInt("database.max_open_conns"),
		MaxIdleConns:    v.GetInt("database.max_idle_conns"),
		ConnMaxLifetime: v.GetDuration("database.conn_max_lifetime"),
	}

	cfg.Redis = RedisConfig{
		URL:      v.GetString("redis.url"),
		Password: v.GetString("redis.password"),
		DB:       v.GetInt("redis.db"),
	}

	cfg.Security = SecurityConfig{
		KMSKeyARN:              v.GetString("security.kms_key_arn"),
		PlayIntegrityProjectID: v.GetString("security.play_integrity_project_id"),
		TxTimeWindowMinutes:    v.GetInt("security.tx_time_window_minutes"),
		IdempotencyTTLSeconds:  v.GetInt("security.idempotency_ttl_seconds"),
	}

	cfg.Limits = LimitsConfig{
		RateLimitPerDevice: v.GetInt("limits.rate_limit_per_device"),
		RateLimitPerWallet: v.GetInt("limits.rate_limit_per_wallet"),
		RateLimitPerIP:     v.GetInt("limits.rate_limit_per_ip"),
	}

	// Validate required fields
	if cfg.Database.URL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.Redis.URL == "" {
		cfg.Redis.URL = "redis://localhost:6379"
	}

	slog.Info("Configuration loaded",
		"port", cfg.Server.Port,
		"db_max_conns", cfg.Database.MaxOpenConns,
	)

	return cfg, nil
}

// IsDevelopment returns true if running in development mode
func IsDevelopment() bool {
	env := os.Getenv("ATHEER_ENV")
	return env == "" || env == "development"
}

// LogLevel returns the configured log level
func LogLevel() slog.Level {
	level := os.Getenv("LOG_LEVEL")
	switch strings.ToUpper(level) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// SetupLogger configures the global slog logger
func SetupLogger() {
	opts := &slog.HandlerOptions{
		Level: LogLevel(),
	}

	var handler slog.Handler
	if IsDevelopment() {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("Logger initialized",
		"level", LogLevel().String(),
		"env", fmt.Sprintf("development=%v", IsDevelopment()),
	)
}
