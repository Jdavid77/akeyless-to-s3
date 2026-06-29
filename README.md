# Akeyless to S3 Exporter

Exports secrets from Akeyless to S3 as a single zip archive, preserving folder structure.

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
- `S3_PREFIX` - Key prefix inside the bucket (default: bucket root). E.g. `backups` → `backups/secrets-20260628-213921.zip`
- `ZIP_PASSWORD` - AES-256 password to encrypt the zip archive. If not set, the zip is uploaded unencrypted and a warning is logged.
- `LOG_LEVEL` - Log level: debug, info, warn, error (default: `info`)
- `LOG_FORMAT` - Log format: console, json (default: `console`)

## Output

Each run produces a single zip file uploaded to S3:

```
s3://<bucket>/<S3_PREFIX>/secrets-<YYYYMMDD-HHMMSS>.zip
```

The zip preserves the Akeyless folder structure. Each secret is stored as a JSON file:

```
app/
  db/
    password.json
  api/
    key.json
infra/
  redis/
    url.json
```

Each `.json` file contains:

```json
{
  "name": "password",
  "path": "/app/db/password",
  "value": "...",
  "timestamp": "2026-06-28T21:39:21Z"
}
```

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
