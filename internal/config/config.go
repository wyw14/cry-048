// Package config loads application configuration from environment and .env.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration.
type Config struct {
	HTTP     HTTPConfig
	Postgres PostgresConfig
	Storage  StorageConfig
	Log      LogConfig
	CORS     CORSConfig
	Runtime  RuntimeConfig
}

type HTTPConfig struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

type PostgresConfig struct {
	DSN             string
	MaxConns        int32
	MinConns        int32
	ConnMaxLifetime time.Duration
	QueryTimeout    time.Duration
}

type StorageConfig struct {
	BaseDir      string
	MaxBytes     int64
	AllowedTypes []string
}

type LogConfig struct {
	Level  string
	Format string
	Output string
}

type CORSConfig struct {
	AllowOrigins []string
	AllowHeaders []string
	AllowMethods []string
}

type RuntimeConfig struct {
	Mode          string // memory | postgres
	SeedOnStartup bool
	JWTSecret     string
	DefaultUserID string
}

// Load reads configuration from environment with defaults applied.
func Load() (*Config, error) {
	cfg := &Config{
		HTTP: HTTPConfig{
			Addr:            getenv("HTTP_ADDR", ":8080"),
			ReadTimeout:     getduration("HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getduration("HTTP_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:     getduration("HTTP_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getduration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		Postgres: PostgresConfig{
			DSN:             getenv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/design_review?sslmode=disable"),
			MaxConns:        getenvInt32("POSTGRES_MAX_CONNS", 10),
			MinConns:        getenvInt32("POSTGRES_MIN_CONNS", 2),
			ConnMaxLifetime: getduration("POSTGRES_CONN_MAX_LIFETIME", time.Hour),
			QueryTimeout:    getduration("POSTGRES_QUERY_TIMEOUT", 5*time.Second),
		},
		Storage: StorageConfig{
			BaseDir:      getenv("STORAGE_BASE_DIR", "./var/attachments"),
			MaxBytes:     getenvInt64("STORAGE_MAX_BYTES", 20*1024*1024),
			AllowedTypes: strings.Split(getenv("STORAGE_ALLOWED_TYPES", "image/png,image/jpeg,image/webp,image/gif,application/pdf,text/plain"), ","),
		},
		Log: LogConfig{
			Level:  getenv("LOG_LEVEL", "info"),
			Format: getenv("LOG_FORMAT", "json"),
			Output: getenv("LOG_OUTPUT", "stdout"),
		},
		CORS: CORSConfig{
			AllowOrigins: strings.Split(getenv("CORS_ALLOW_ORIGINS", "http://localhost:5173,http://localhost:8080"), ","),
			AllowHeaders: strings.Split(getenv("CORS_ALLOW_HEADERS", "Origin,Content-Type,Accept,Authorization,X-Request-ID"), ","),
			AllowMethods: strings.Split(getenv("CORS_ALLOW_METHODS", "GET,POST,PUT,DELETE,OPTIONS,PATCH"), ","),
		},
		Runtime: RuntimeConfig{
			Mode:          getenv("RUNTIME_MODE", "memory"),
			SeedOnStartup: getenvBool("SEED_ON_STARTUP", true),
			JWTSecret:     getenv("JWT_SECRET", "local-dev-secret"),
			DefaultUserID: getenv("DEFAULT_USER_ID", "user-local"),
		},
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate checks the configuration.
func (c *Config) Validate() error {
	if c.Runtime.Mode != "memory" && c.Runtime.Mode != "postgres" {
		return errors.New("RUNTIME_MODE must be 'memory' or 'postgres'")
	}
	if c.Storage.MaxBytes <= 0 {
		return errors.New("STORAGE_MAX_BYTES must be positive")
	}
	if c.Postgres.MaxConns < c.Postgres.MinConns {
		return errors.New("POSTGRES_MAX_CONNS must be >= POSTGRES_MIN_CONNS")
	}
	return nil
}

// AllowedTypesMap returns the allowed types as a set.
func (c *Config) AllowedTypesMap() map[string]bool {
	m := make(map[string]bool, len(c.Storage.AllowedTypes))
	for _, t := range c.Storage.AllowedTypes {
		t = strings.ToLower(strings.TrimSpace(t))
		if t != "" {
			m[t] = true
		}
	}
	return m
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt32(key string, def int32) int32 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil {
			return int32(n)
		}
	}
	return def
}

func getenvInt64(key string, def int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return def
}

func getenvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func getduration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
		if n, err := strconv.Atoi(v); err == nil {
			return time.Duration(n) * time.Second
		}
	}
	return def
}

// String returns a debug representation with secrets masked.
func (c *Config) String() string {
	return fmt.Sprintf("Config{HTTP.Addr=%s, Mode=%s, PG.DSN=%s, Storage=%s, Seed=%v}",
		c.HTTP.Addr, c.Runtime.Mode, maskDSN(c.Postgres.DSN), c.Storage.BaseDir, c.Runtime.SeedOnStartup)
}

func maskDSN(dsn string) string {
	// mask password in DSN if present
	idx := strings.Index(dsn, "://")
	if idx < 0 {
		return dsn
	}
	scheme := dsn[:idx+3]
	rest := dsn[idx+3:]
	at := strings.LastIndex(rest, "@")
	if at < 0 {
		return dsn
	}
	userinfo := rest[:at]
	host := rest[at:]
	colon := strings.Index(userinfo, ":")
	if colon < 0 {
		return dsn
	}
	return scheme + userinfo[:colon] + ":***" + host
}
