package s3uploader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Jdavid77/akeylesstos3/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog/log"
)

const (
	maxRetries     = 3
	retryDelay     = 2 * time.Second
	maxConcurrency = 10
)

// Uploader handles uploading secrets to S3
type Uploader struct {
	client *s3.Client
	bucket string
}

// NewUploader creates a new S3 uploader
func NewUploader(region, accessKeyID, secretAccessKey, bucket, endpoint string) (*Uploader, error) {
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

	// Create S3 client with optional custom endpoint
	var client *s3.Client
	if endpoint != "" {
		client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true // Required for MinIO and some S3-compatible services
		})
		log.Info().Str("bucket", bucket).Str("region", region).Str("endpoint", endpoint).Msg("S3 uploader initialized with custom endpoint")
	} else {
		client = s3.NewFromConfig(cfg)
		log.Info().Str("bucket", bucket).Str("region", region).Msg("S3 uploader initialized")
	}

	return &Uploader{
		client: client,
		bucket: bucket,
	}, nil
}

// UploadResult represents the result of an upload operation
type UploadResult struct {
	SecretPath string
	S3Key      string
	Success    bool
	Error      error
}

// UploadSecrets uploads multiple secrets to S3 concurrently
func (u *Uploader) UploadSecrets(ctx context.Context, secrets []*models.Secret) []UploadResult {
	log.Info().Int("total_secrets", len(secrets)).Msg("Starting concurrent upload of secrets")

	results := make([]UploadResult, len(secrets))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxConcurrency)

	for i, secret := range secrets {
		wg.Add(1)
		go func(idx int, s *models.Secret) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := u.uploadSecretWithRetry(ctx, s)
			results[idx] = result
		}(i, secret)
	}

	wg.Wait()
	close(semaphore)

	// Count successes and failures
	successes := 0
	failures := 0
	for _, result := range results {
		if result.Success {
			successes++
		} else {
			failures++
		}
	}

	log.Info().
		Int("total", len(secrets)).
		Int("successes", successes).
		Int("failures", failures).
		Msg("Completed uploading secrets")

	return results
}

// uploadSecretWithRetry uploads a single secret with retry logic
func (u *Uploader) uploadSecretWithRetry(ctx context.Context, secret *models.Secret) UploadResult {
	s3Key := u.buildS3Key(secret.Path)

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := u.uploadSecret(ctx, secret, s3Key)
		if err == nil {
			log.Info().
				Str("secret_path", secret.Path).
				Str("s3_key", s3Key).
				Msg("Successfully uploaded secret")
			return UploadResult{
				SecretPath: secret.Path,
				S3Key:      s3Key,
				Success:    true,
			}
		}

		lastErr = err
		if attempt < maxRetries {
			log.Warn().
				Err(err).
				Str("secret_path", secret.Path).
				Int("attempt", attempt).
				Int("max_retries", maxRetries).
				Msg("Upload failed, retrying")
			time.Sleep(retryDelay * time.Duration(attempt))
		}
	}

	log.Error().
		Err(lastErr).
		Str("secret_path", secret.Path).
		Str("s3_key", s3Key).
		Msg("Failed to upload secret after all retries")

	return UploadResult{
		SecretPath: secret.Path,
		S3Key:      s3Key,
		Success:    false,
		Error:      lastErr,
	}
}

// uploadSecret uploads a single secret to S3
func (u *Uploader) uploadSecret(ctx context.Context, secret *models.Secret, s3Key string) error {
	// Marshal secret to JSON
	jsonData, err := json.MarshalIndent(secret, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal secret to JSON: %w", err)
	}

	// Upload to S3
	_, err = u.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(u.bucket),
		Key:         aws.String(s3Key),
		Body:        bytes.NewReader(jsonData),
		ContentType: aws.String("application/json"),
	})

	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

// buildS3Key constructs the S3 key from the secret path
// Preserves the folder structure and adds .json extension
func (u *Uploader) buildS3Key(secretPath string) string {
	// Remove leading slash if present
	key := strings.TrimPrefix(secretPath, "/")

	// Ensure .json extension
	if !strings.HasSuffix(key, ".json") {
		key = key + ".json"
	}

	return key
}
