package config

import (
	"strings"
	"testing"
)

// requiredVars lists every env var that must be present for Load to succeed.
var requiredVars = map[string]string{
	"AKEYLESS_ACCESS_ID":     "id",
	"AKEYLESS_ACCESS_KEY":    "key",
	"AKEYLESS_GATEWAY_URL":   "https://gateway",
	"AWS_REGION":             "us-east-1",
	"AWS_ACCESS_KEY_ID":      "accesskey",
	"AWS_SECRET_ACCESS_KEY":  "secretkey",
	"S3_BUCKET":              "my-bucket",
	"S3_ENDPOINT":            "https://s3.example.com",
}

func setEnv(t *testing.T, vars map[string]string) {
	t.Helper()
	for k, v := range vars {
		t.Setenv(k, v)
	}
}

func TestLoad_success(t *testing.T) {
	setEnv(t, requiredVars)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.AkeylessAccessID != "id" {
		t.Errorf("AkeylessAccessID = %q, want %q", cfg.AkeylessAccessID, "id")
	}
	if cfg.BasePath != "/" {
		t.Errorf("BasePath = %q, want default %q", cfg.BasePath, "/")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want default %q", cfg.LogLevel, "info")
	}
}

func TestLoad_missingRequiredVars(t *testing.T) {
	for missing := range requiredVars {
		t.Run("missing_"+missing, func(t *testing.T) {
			for k, v := range requiredVars {
				if k != missing {
					t.Setenv(k, v)
				}
			}
			_, err := Load()
			if err == nil {
				t.Fatal("expected error when required var is missing, got nil")
			}
			if !strings.Contains(err.Error(), missing) {
				t.Errorf("error %q should mention missing var %q", err.Error(), missing)
			}
		})
	}
}

func TestLoad_optionalDefaults(t *testing.T) {
	setEnv(t, requiredVars)
	// BASE_PATH and LOG_LEVEL and LOG_FORMAT are not set — defaults should apply.

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BasePath != "/" {
		t.Errorf("BasePath = %q, want %q", cfg.BasePath, "/")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.LogFormat != "console" {
		t.Errorf("LogFormat = %q, want %q", cfg.LogFormat, "console")
	}
}
