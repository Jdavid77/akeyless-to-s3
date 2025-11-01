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

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger.Init(cfg.LogLevel)
	log.Info().Msg("Starting Akeyless to S3 exporter")

	// Run the main application logic
	if err := run(cfg); err != nil {
		log.Fatal().Err(err).Msg("Application failed")
	}

	log.Info().Msg("Application completed successfully")
}

func run(cfg *config.Config) error {
	ctx := context.Background()

	// Initialize Akeyless client
	log.Info().Msg("Initializing Akeyless client")
	akeylessClient, err := akeyless.NewClient(
		cfg.AkeylessGatewayURL,
		cfg.AkeylessAccessID,
		cfg.AkeylessAccessKey,
		cfg.BasePath,
	)
	if err != nil {
		return fmt.Errorf("failed to create Akeyless client: %w", err)
	}

	// List all secrets
	log.Info().Msg("Listing all secrets from Akeyless")
	secretItems, err := akeylessClient.ListAllSecrets(ctx)
	if err != nil {
		return fmt.Errorf("failed to list secrets: %w", err)
	}

	if len(secretItems) == 0 {
		log.Warn().Msg("No secrets found in Akeyless")
		return nil
	}

	log.Info().Int("count", len(secretItems)).Msg("Found secrets")

	// Retrieve secret values
	log.Info().Msg("Retrieving secret values")
	secrets := make([]*models.Secret, 0, len(secretItems))
	failedRetrievals := 0

	for _, item := range secretItems {
		secret, err := akeylessClient.GetSecretValue(ctx, item.ItemName)
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

	// Initialize S3 uploader
	log.Info().Msg("Initializing S3 uploader")
	uploader, err := s3uploader.NewUploader(
		cfg.AWSRegion,
		cfg.AWSAccessKeyID,
		cfg.AWSSecretAccessKey,
		cfg.S3Bucket,
		cfg.S3Endpoint,
	)
	if err != nil {
		return fmt.Errorf("failed to create S3 uploader: %w", err)
	}

	// Upload secrets to S3
	log.Info().Msg("Uploading secrets to S3")
	results := uploader.UploadSecrets(ctx, secrets)

	// Analyze results
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
