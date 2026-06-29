package s3uploader

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/Jdavid77/akeylesstos3/internal/models"
)

func TestBuildS3Key(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/app/db/password", "app/db/password.json"},
		{"/production/db/password", "production/db/password.json"},
		{"app/db/password", "app/db/password.json"},
		{"/app/db/secret.json", "app/db/secret.json"},
		{"/password", "password.json"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := BuildS3Key(tt.path); got != tt.want {
				t.Errorf("BuildS3Key(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestBuildZipKey(t *testing.T) {
	ts := time.Date(2026, 6, 28, 21, 39, 21, 0, time.UTC)

	tests := []struct {
		prefix string
		want   string
	}{
		{"", "secrets-20260628-213921.zip"},
		{"backups", "backups/secrets-20260628-213921.zip"},
		{"backups/", "backups/secrets-20260628-213921.zip"},
		{"env/prod", "env/prod/secrets-20260628-213921.zip"},
	}

	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			if got := BuildZipKey(tt.prefix, ts); got != tt.want {
				t.Errorf("BuildZipKey(%q) = %q, want %q", tt.prefix, got, tt.want)
			}
		})
	}
}

func TestBuildZip_noPassword(t *testing.T) {
	secrets := []*models.Secret{
		{Name: "password", Path: "/app/db/password", Value: "s3cr3t"},
		{Name: "key", Path: "/app/api/key", Value: "apikey"},
	}

	data, err := BuildZip(secrets, "")
	if err != nil {
		t.Fatalf("BuildZip returned error: %v", err)
	}

	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("zip.NewReader failed: %v", err)
	}

	if len(r.File) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(r.File))
	}

	wantNames := map[string]bool{
		"app/db/password.json": true,
		"app/api/key.json":     true,
	}
	for _, f := range r.File {
		if !wantNames[f.Name] {
			t.Errorf("unexpected zip entry %q", f.Name)
		}
	}
}

func TestBuildZip_withPassword(t *testing.T) {
	secrets := []*models.Secret{
		{Name: "password", Path: "/app/db/password", Value: "s3cr3t"},
	}

	data, err := BuildZip(secrets, "hunter2")
	if err != nil {
		t.Fatalf("BuildZip returned error: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("BuildZip returned empty data")
	}

	// The zip is valid but entries are encrypted — verify it's a valid zip.
	_, err = zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("encrypted zip is not a valid zip archive: %v", err)
	}
}

func TestBuildZip_contentRoundtrip(t *testing.T) {
	secret := &models.Secret{Name: "key", Path: "/svc/api/key", Value: "myvalue"}

	data, err := BuildZip([]*models.Secret{secret}, "")
	if err != nil {
		t.Fatalf("BuildZip error: %v", err)
	}

	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("zip.NewReader error: %v", err)
	}

	rc, err := r.File[0].Open()
	if err != nil {
		t.Fatalf("open zip entry: %v", err)
	}
	defer rc.Close()

	var got models.Secret
	if err := json.NewDecoder(rc).Decode(&got); err != nil {
		t.Fatalf("decode zip entry: %v", err)
	}

	if got.Value != secret.Value {
		t.Errorf("value = %q, want %q", got.Value, secret.Value)
	}
	if got.Path != secret.Path {
		t.Errorf("path = %q, want %q", got.Path, secret.Path)
	}
}
