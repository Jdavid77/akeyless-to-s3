# Akeyless to S3 Exporter

Exports secrets from Akeyless to S3 as JSON files, preserving folder structure.

## Configuration

Copy `.env.example` to `.env` and configure:

**Required:**
- `AKEYLESS_ACCESS_ID` - Akeyless Access ID
- `AKEYLESS_ACCESS_KEY` - Akeyless Access Key
- `AKEYLESS_GATEWAY_URL` - Akeyless Gateway URL
- `AWS_REGION` - AWS Region
- `AWS_ACCESS_KEY_ID` - AWS Access Key
- `AWS_SECRET_ACCESS_KEY` - AWS Secret Key
- `S3_BUCKET` - S3 Bucket name
- `S3_ENDPOINT` - S3 endpoint URL (AWS S3, MinIO, DigitalOcean Spaces, etc.)

**Optional:**
- `BASE_PATH` - Base path to export from (default: `/`)
- `LOG_LEVEL` - Log level: debug, info, warn, error (default: `info`)
- `LOG_FORMAT` - Log format: console, json (default: `console`)

## Usage

**Local:**
```bash
export $(cat .env | xargs)
go run main.go
```

**Docker Compose:**
```bash
docker-compose up --build
```

**Docker:**
```bash
docker build -t akeylesstos3 .
docker run --env-file .env akeylesstos3
```
