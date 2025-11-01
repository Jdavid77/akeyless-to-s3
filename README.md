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

## Output

Secrets are stored as `path/to/secret.json`:
```json
{
  "name": "secret-name",
  "path": "/original/path",
  "value": "secret-value",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## IAM Permissions

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": ["s3:PutObject", "s3:PutObjectAcl"],
    "Resource": "arn:aws:s3:::your-bucket/*"
  }]
}
```

## Releases

This project uses [semantic-release](https://github.com/semantic-release/semantic-release) for automated versioning and releases.

**Commit Message Format:**
- `feat:` - New feature (triggers minor version bump)
- `fix:` - Bug fix (triggers patch version bump)
- `docs:` - Documentation changes
- `chore:` - Maintenance tasks
- `BREAKING CHANGE:` - Breaking changes (triggers major version bump)

**Examples:**
```
feat: add support for custom retry configuration
fix: resolve connection timeout issue with S3
docs: update README with new configuration options
```

Releases are automatically created when commits are pushed to the `main` branch.
