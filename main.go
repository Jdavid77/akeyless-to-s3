package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Jdavid77/akeylesstos3/internal/akeyless"
	"github.com/Jdavid77/akeylesstos3/internal/config"
	"github.com/Jdavid77/akeylesstos3/internal/logger"
	"github.com/Jdavid77/akeylesstos3/internal/models"
	"github.com/Jdavid77/akeylesstos3/internal/s3uploader"
	"github.com/rs/zerolog/log"
)

// SecretSource lists and retrieves secrets from a secrets manager.
type SecretSource interface {
	ListAllSecrets(ctx context.Context) ([]models.SecretItem, error)
	GetSecretValue(ctx context.Context, secretPath string) (*models.Secret, error)
}

// SecretSink uploads secrets to a destination store.
type SecretSink interface {
	UploadSecrets(ctx context.Context, secrets []*models.Secret) []s3uploader.UploadResult
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	l := logger.Init(cfg.LogLevel, cfg.LogFormat)
	log.Info().Msg("Starting Akeyless to S3 exporter")

	akeylessClient, err := akeyless.NewClient(
		cfg.AkeylessGatewayURL,
		cfg.AkeylessAccessID,
		cfg.AkeylessAccessKey,
		cfg.BasePath,
		l,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Akeyless client")
	}

	uploader, err := s3uploader.NewUploader(
		cfg.AWSRegion,
		cfg.AWSAccessKeyID,
		cfg.AWSSecretAccessKey,
		cfg.S3Bucket,
		cfg.S3Endpoint,
		cfg.S3Prefix,
		cfg.ZipPassword,
		l,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize S3 uploader")
	}

	if err := run(akeylessClient, uploader); err != nil {
		log.Fatal().Err(err).Msg("Application failed")
	}

	log.Info().Msg("Application completed successfully")
}

func run(src SecretSource, sink SecretSink) error {
	ctx := context.Background()

	log.Info().Msg("Listing all secrets from Akeyless")
	secretItems, err := src.ListAllSecrets(ctx)
	if err != nil {
		return fmt.Errorf("failed to list secrets: %w", err)
	}

	if len(secretItems) == 0 {
		log.Warn().Msg("No secrets found in Akeyless")
		return nil
	}

	log.Info().Int("count", len(secretItems)).Msg("Found secrets")

	log.Info().Msg("Retrieving secret values")
	secrets := make([]*models.Secret, 0, len(secretItems))
	failedRetrievals := 0

	for _, item := range secretItems {
		secret, err := src.GetSecretValue(ctx, item.ItemName)
		if err != nil {
			log.Error().
				Err(err).
				Str("secret_path", item.ItemName).
				Msg("Failed to retrieve secret value")
			failedRetrievals++
			continue
		}
		secrets = append(secrets, secret)
	}

	log.Info().
		Int("retrieved", len(secrets)).
		Int("failed", failedRetrievals).
		Msg("Finished retrieving secret values")

	if len(secrets) == 0 {
		return fmt.Errorf("no secrets could be retrieved")
	}

	log.Info().Msg("Uploading secrets to S3")
	results := sink.UploadSecrets(ctx, secrets)

	failedUploads := 0
	for _, result := range results {
		if !result.Success {
			failedUploads++
		}
	}

	if failedUploads > 0 {
		log.Warn().
			Int("failed_uploads", failedUploads).
			Int("total", len(results)).
			Msg("Some uploads failed")
		return fmt.Errorf("%d out of %d uploads failed", failedUploads, len(results))
	}

	log.Info().
		Int("total_secrets", len(secrets)).
		Msg("Successfully exported all secrets to S3")

	return nil
}
