package main

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/Jdavid77/akeylesstos3/internal/models"
	"github.com/Jdavid77/akeylesstos3/internal/s3uploader"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestMain(m *testing.M) {
	log.Logger = zerolog.Nop()
	os.Exit(m.Run())
}

// fakeSource is a test double for SecretSource.
type fakeSource struct {
	listFn func(ctx context.Context) ([]models.SecretItem, error)
	getFn  func(ctx context.Context, path string) (*models.Secret, error)
}

func (f *fakeSource) ListAllSecrets(ctx context.Context) ([]models.SecretItem, error) {
	return f.listFn(ctx)
}

func (f *fakeSource) GetSecretValue(ctx context.Context, path string) (*models.Secret, error) {
	return f.getFn(ctx, path)
}

// fakeSink is a test double for SecretSink.
type fakeSink struct {
	uploadFn func(ctx context.Context, secrets []*models.Secret) []s3uploader.UploadResult
}

func (f *fakeSink) UploadSecrets(ctx context.Context, secrets []*models.Secret) []s3uploader.UploadResult {
	return f.uploadFn(ctx, secrets)
}

// successSink returns a successful UploadResult for every secret.
func successSink() *fakeSink {
	return &fakeSink{
		uploadFn: func(_ context.Context, secrets []*models.Secret) []s3uploader.UploadResult {
			results := make([]s3uploader.UploadResult, len(secrets))
			for i, s := range secrets {
				results[i] = s3uploader.UploadResult{SecretPath: s.Path, S3Key: s.Path + ".json", Success: true}
			}
			return results
		},
	}
}

var oneItem = []models.SecretItem{{ItemName: "/app/db/password", ItemType: "static-secret"}}

func alwaysGetSecret(_ context.Context, path string) (*models.Secret, error) {
	return &models.Secret{Name: "password", Path: path, Value: "s3cr3t"}, nil
}

func TestRun(t *testing.T) {
	tests := []struct {
		name    string
		src     SecretSource
		sink    SecretSink
		wantErr string // empty string means no error expected
	}{
		{
			name: "list error returns error",
			src: &fakeSource{
				listFn: func(_ context.Context) ([]models.SecretItem, error) {
					return nil, errors.New("network timeout")
				},
			},
			sink:    successSink(),
			wantErr: "failed to list secrets",
		},
		{
			name: "empty list returns nil",
			src: &fakeSource{
				listFn: func(_ context.Context) ([]models.SecretItem, error) {
					return []models.SecretItem{}, nil
				},
			},
			sink:    successSink(),
			wantErr: "",
		},
		{
			name: "all retrievals fail returns error",
			src: &fakeSource{
				listFn: func(_ context.Context) ([]models.SecretItem, error) { return oneItem, nil },
				getFn: func(_ context.Context, _ string) (*models.Secret, error) {
					return nil, errors.New("access denied")
				},
			},
			sink:    successSink(),
			wantErr: "no secrets could be retrieved",
		},
		{
			name: "partial retrieval failure continues and succeeds",
			src: &fakeSource{
				listFn: func(_ context.Context) ([]models.SecretItem, error) {
					return []models.SecretItem{
						{ItemName: "/app/a", ItemType: "static-secret"},
						{ItemName: "/app/b", ItemType: "static-secret"},
					}, nil
				},
				getFn: func(_ context.Context, path string) (*models.Secret, error) {
					if path == "/app/a" {
						return nil, errors.New("access denied")
					}
					return &models.Secret{Name: "b", Path: path, Value: "val"}, nil
				},
			},
			sink:    successSink(),
			wantErr: "",
		},
		{
			name: "upload failures returns error with count",
			src: &fakeSource{
				listFn: func(_ context.Context) ([]models.SecretItem, error) { return oneItem, nil },
				getFn:  alwaysGetSecret,
			},
			sink: &fakeSink{
				uploadFn: func(_ context.Context, secrets []*models.Secret) []s3uploader.UploadResult {
					return []s3uploader.UploadResult{{
						SecretPath: secrets[0].Path,
						Success:    false,
						Error:      errors.New("S3 unavailable"),
					}}
				},
			},
			wantErr: "1 out of 1 uploads failed",
		},
		{
			name: "happy path returns nil",
			src: &fakeSource{
				listFn: func(_ context.Context) ([]models.SecretItem, error) { return oneItem, nil },
				getFn:  alwaysGetSecret,
			},
			sink:    successSink(),
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := run(tt.src, tt.sink)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Errorf("expected error containing %q, got nil", tt.wantErr)
				return
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got: %q", tt.wantErr, err.Error())
			}
		})
	}
}
