package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds all configuration for the application
type Config struct {
	// Akeyless configuration
	AkeylessAccessID   string
	AkeylessAccessKey  string
	AkeylessGatewayURL string
	BasePath           string

	// AWS S3 configuration
	AWSRegion          string
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	S3Bucket           string
	S3Endpoint         string

	// Logging configuration
	LogLevel  string
	LogFormat string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		AkeylessAccessID:   os.Getenv("AKEYLESS_ACCESS_ID"),
		AkeylessAccessKey:  os.Getenv("AKEYLESS_ACCESS_KEY"),
		AkeylessGatewayURL: os.Getenv("AKEYLESS_GATEWAY_URL"),
		BasePath:           getEnvOrDefault("BASE_PATH", "/"),
		AWSRegion:          os.Getenv("AWS_REGION"),
		AWSAccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		AWSSecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		S3Bucket:           os.Getenv("S3_BUCKET"),
		S3Endpoint:         os.Getenv("S3_ENDPOINT"),
		LogLevel:           getEnvOrDefault("LOG_LEVEL", "info"),
		LogFormat:          getEnvOrDefault("LOG_FORMAT", "console"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that all required configuration is present
func (c *Config) Validate() error {
	var missing []string

	if c.AkeylessAccessID == "" {
		missing = append(missing, "AKEYLESS_ACCESS_ID")
	}
	if c.AkeylessAccessKey == "" {
		missing = append(missing, "AKEYLESS_ACCESS_KEY")
	}
	if c.AkeylessGatewayURL == "" {
		missing = append(missing, "AKEYLESS_GATEWAY_URL")
	}
	if c.AWSRegion == "" {
		missing = append(missing, "AWS_REGION")
	}
	if c.AWSAccessKeyID == "" {
		missing = append(missing, "AWS_ACCESS_KEY_ID")
	}
	if c.AWSSecretAccessKey == "" {
		missing = append(missing, "AWS_SECRET_ACCESS_KEY")
	}
	if c.S3Bucket == "" {
		missing = append(missing, "S3_BUCKET")
	}
	if c.S3Endpoint == "" {
		missing = append(missing, "S3_ENDPOINT")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
