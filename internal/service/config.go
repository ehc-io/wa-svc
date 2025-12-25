package service

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the WhatsApp API service.
type Config struct {
	// Server settings
	Host string
	Port int

	// Data directory (where session.db, wacli.db, media/ live)
	DataDir string

	// API authentication
	APIKey string

	// Webhook settings
	WebhookURL     string
	WebhookSecret  string
	WebhookRetries int
	WebhookTimeout time.Duration

	// Sync settings
	DownloadMedia   bool
	RefreshContacts bool
	RefreshGroups   bool

	// Graceful shutdown timeout
	ShutdownTimeout time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Host:            "0.0.0.0",
		Port:            8080,
		DataDir:         "/data",
		WebhookRetries:  3,
		WebhookTimeout:  10 * time.Second,
		DownloadMedia:   true,
		RefreshContacts: true,
		RefreshGroups:   true,
		ShutdownTimeout: 30 * time.Second,
	}
}

// LoadFromEnv loads configuration from environment variables.
func LoadFromEnv() Config {
	cfg := DefaultConfig()

	if v := os.Getenv("WASVC_HOST"); v != "" {
		cfg.Host = v
	}
	if v := os.Getenv("WASVC_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil && port > 0 {
			cfg.Port = port
		}
	}
	if v := os.Getenv("WASVC_DATA_DIR"); v != "" {
		cfg.DataDir = v
	}
	if v := os.Getenv("WASVC_API_KEY"); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv("WASVC_WEBHOOK_URL"); v != "" {
		cfg.WebhookURL = v
	}
	if v := os.Getenv("WASVC_WEBHOOK_SECRET"); v != "" {
		cfg.WebhookSecret = v
	}
	if v := os.Getenv("WASVC_WEBHOOK_RETRIES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			cfg.WebhookRetries = n
		}
	}
	if v := os.Getenv("WASVC_WEBHOOK_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.WebhookTimeout = d
		}
	}
	if v := os.Getenv("WASVC_DOWNLOAD_MEDIA"); v != "" {
		cfg.DownloadMedia = parseBool(v, true)
	}
	if v := os.Getenv("WASVC_REFRESH_CONTACTS"); v != "" {
		cfg.RefreshContacts = parseBool(v, true)
	}
	if v := os.Getenv("WASVC_REFRESH_GROUPS"); v != "" {
		cfg.RefreshGroups = parseBool(v, true)
	}
	if v := os.Getenv("WASVC_SHUTDOWN_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.ShutdownTimeout = d
		}
	}

	return cfg
}

// Validate checks that the configuration is valid.
func (c Config) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}
	if strings.TrimSpace(c.DataDir) == "" {
		return fmt.Errorf("data directory is required")
	}
	return nil
}

// Addr returns the address to listen on.
func (c Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func parseBool(s string, defaultVal bool) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return defaultVal
	}
}
