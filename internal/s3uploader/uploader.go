package s3uploader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	alexzip "github.com/alexmullins/zip"
	"github.com/Jdavid77/akeylesstos3/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog"
)

const (
	maxRetries = 3
	retryDelay = 2 * time.Second
)

// Uploader handles uploading secrets to S3 as a single zip archive.
type Uploader struct {
	client   *s3.Client
	bucket   string
	prefix   string
	password string
	log      zerolog.Logger
}

// NewUploader creates a new S3 uploader.
func NewUploader(region, accessKeyID, secretAccessKey, bucket, endpoint, prefix, password string, log zerolog.Logger) (*Uploader, error) {
	cfgOpts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKeyID,
			secretAccessKey,
			"",
		)),
	}

	cfg, err := config.LoadDefaultConfig(context.Background(), cfgOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	var client *s3.Client
	if endpoint != "" {
		client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		})
		log.Info().Str("bucket", bucket).Str("region", region).Str("endpoint", endpoint).Msg("S3 uploader initialized with custom endpoint")
	} else {
		client = s3.NewFromConfig(cfg)
		log.Info().Str("bucket", bucket).Str("region", region).Msg("S3 uploader initialized")
	}

	return &Uploader{
		client:   client,
		bucket:   bucket,
		prefix:   prefix,
		password: password,
		log:      log,
	}, nil
}

// UploadResult represents the result of an upload operation.
type UploadResult struct {
	SecretPath string
	S3Key      string
	Success    bool
	Error      error
}

// UploadSecrets zips all secrets into a single archive and uploads it to S3.
func (u *Uploader) UploadSecrets(ctx context.Context, secrets []*models.Secret) []UploadResult {
	if u.password == "" {
		u.log.Warn().Msg("ZIP_PASSWORD not set — uploading unencrypted zip")
	}

	zipKey := BuildZipKey(u.prefix, time.Now().UTC())
	u.log.Info().Str("zip_key", zipKey).Int("secrets", len(secrets)).Msg("Building zip archive")

	zipData, err := BuildZip(secrets, u.password)
	if err != nil {
		u.log.Error().Err(err).Msg("Failed to build zip archive")
		return []UploadResult{{S3Key: zipKey, Success: false, Error: err}}
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		_, err = u.client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(u.bucket),
			Key:         aws.String(zipKey),
			Body:        bytes.NewReader(zipData),
			ContentType: aws.String("application/zip"),
		})
		if err == nil {
			u.log.Info().Str("zip_key", zipKey).Msg("Successfully uploaded zip archive")
			return []UploadResult{{S3Key: zipKey, Success: true}}
		}
		lastErr = err
		if attempt < maxRetries {
			u.log.Warn().Err(err).Int("attempt", attempt).Int("max_retries", maxRetries).Msg("Upload failed, retrying")
			time.Sleep(retryDelay * time.Duration(attempt))
		}
	}

	u.log.Error().Err(lastErr).Str("zip_key", zipKey).Msg("Failed to upload zip archive after all retries")
	return []UploadResult{{S3Key: zipKey, Success: false, Error: lastErr}}
}

// BuildZipKey returns the S3 key for the zip archive.
// prefix may be empty (zip lands at bucket root) or a path prefix like "backups".
func BuildZipKey(prefix string, t time.Time) string {
	name := "secrets-" + t.Format("20060102-150405") + ".zip"
	if prefix == "" {
		return name
	}
	return strings.TrimSuffix(prefix, "/") + "/" + name
}

// BuildZip creates an in-memory zip archive from secrets.
// If password is non-empty each entry is AES-256 encrypted.
func BuildZip(secrets []*models.Secret, password string) ([]byte, error) {
	var buf bytes.Buffer
	w := alexzip.NewWriter(&buf)

	for _, secret := range secrets {
		jsonData, err := json.MarshalIndent(secret, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal secret %s: %w", secret.Path, err)
		}

		entryPath := BuildS3Key(secret.Path)

		var fw interface{ Write([]byte) (int, error) }
		if password != "" {
			fw, err = w.Encrypt(entryPath, password)
		} else {
			fw, err = w.Create(entryPath)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to create zip entry for %s: %w", secret.Path, err)
		}

		if _, err = fw.Write(jsonData); err != nil {
			return nil, fmt.Errorf("failed to write zip entry for %s: %w", secret.Path, err)
		}
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize zip: %w", err)
	}

	return buf.Bytes(), nil
}

// BuildS3Key constructs the zip entry path from a secret path.
// Strips the leading slash and appends .json if not already present.
func BuildS3Key(secretPath string) string {
	key := strings.TrimPrefix(secretPath, "/")
	if !strings.HasSuffix(key, ".json") {
		key = key + ".json"
	}
	return key
}
