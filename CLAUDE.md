# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go application that exports secrets from Akeyless to AWS S3, maintaining folder structure and storing each secret as a JSON file.

## Development Commands

### Running the Application

```bash
# Load environment variables from .env file
export $(cat .env | xargs)

# Run with Go (development)
go run main.go

# Build binary
go build -o akeylesstos3 .

# Run compiled binary
./akeylesstos3
```

### Docker

```bash
# Build Docker image
docker build -t akeylesstos3:latest .

# Run container with environment file
docker run --rm --env-file .env akeylesstos3:latest

# Run with Docker Compose
docker-compose up --build
```

### Dependency Management

```bash
# Download dependencies
go mod download

# Update dependencies
go mod tidy

# Verify dependencies
go mod verify
```

## Architecture

The application follows a clean layered architecture with clear separation of concerns:

### Main Flow (main.go)

The application runs as a one-shot job with the following execution flow:

1. Load configuration from environment variables
2. Initialize structured logger (zerolog)
3. Authenticate with Akeyless and obtain token
4. Recursively list all secrets from base path (breadth-first traversal)
5. Retrieve secret values sequentially (continues on individual failures)
6. Upload secrets to S3 concurrently with retry logic
7. Exit with non-zero status if any uploads fail after retries

### Package Structure

**internal/config**: Configuration management
- Loads all settings from environment variables
- Validates required variables on startup
- Required: AKEYLESS_ACCESS_ID, AKEYLESS_ACCESS_KEY, AKEYLESS_GATEWAY_URL, AWS_REGION, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, S3_BUCKET, S3_ENDPOINT
- Optional: BASE_PATH (default: "/"), LOG_LEVEL (default: "info"), LOG_FORMAT (default: "console")

**internal/akeyless**: Akeyless API client wrapper
- Handles authentication using Access ID and Access Key (returns token for subsequent requests)
- Authentication follows official Akeyless Go SDK pattern with both AccessId and AccessKey
- `ListAllSecrets()`: Recursively explores folders using breadth-first search
- Supports secret types: static-secret, dynamic-secret, rotated-secret
- `GetSecretValue()`: Retrieves individual secret values

**internal/s3uploader**: S3 upload handling
- Concurrent uploads with semaphore (maxConcurrency = 10 workers)
- Exponential backoff retry logic (3 retries, 2-second base delay)
- Supports custom S3 endpoints for S3-compatible services (MinIO, DigitalOcean Spaces, etc.)
- Uses path-style addressing when custom endpoint is specified
- Path transformation: `/production/db/password` → `production/db/password.json`

**internal/models**: Data models
- `Secret`: Complete secret with name, path, value, and timestamp
- `SecretItem`: Metadata from Akeyless listing (name and type)

**internal/logger**: Structured logging setup
- Uses zerolog with configurable format (console or JSON)
- Configurable log level via LOG_LEVEL env var (debug, info, warn, error, fatal)
- Configurable log format via LOG_FORMAT env var (console, json)
- Secret values are never logged

### Key Design Decisions

**Error Handling Philosophy**:
- Failed secret retrievals are logged but don't stop the process
- Upload failures trigger retries with exponential backoff
- Application exits with error if any uploads fail after all retries
- Partial success is logged with detailed counts

**Concurrency Model**:
- Secret listing is sequential (recursive folder traversal)
- Secret value retrieval is sequential to avoid overwhelming Akeyless API
- S3 uploads are concurrent (10 workers) with semaphore control
- Thread-safe result collection using goroutines and WaitGroup

**S3 Key Mapping**:
- Leading slash is stripped from Akeyless paths
- Folder structure is preserved exactly
- `.json` extension is automatically appended
- Example: Akeyless `/app/api/key` → S3 `app/api/key.json`

## Environment Configuration

The application requires a `.env` file for local development. Copy `.env.example` to `.env` and fill in actual credentials. The application will exit immediately if any required variables are missing.

## Security Notes

- Secret values are never included in logs
- Docker container runs as non-root user (uid 1000)
- All API communication uses HTTPS
- AWS credentials require minimal IAM permissions: `s3:PutObject`, `s3:PutObjectAcl`
- For production use, enable encryption at rest on your S3 bucket or S3-compatible service
