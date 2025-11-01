# Akeyless to S3 Exporter

A Go application that exports secrets from Akeyless to AWS S3, maintaining the original folder structure and storing each secret as a JSON file.

## Features

- Authenticates with Akeyless using API Key
- Recursively retrieves all secrets from a specified base path
- Preserves the exact folder structure from Akeyless in S3
- Stores each secret as an individual JSON file
- Concurrent uploads with configurable worker pool
- Automatic retry logic with exponential backoff
- Structured JSON logging
- Compatible with S3 and S3-compatible services (MinIO, DigitalOcean Spaces, etc.)
- Designed for Kubernetes CronJob execution

## Prerequisites

- Go 1.21+ (for local development)
- Docker (for containerization)
- Kubernetes cluster (for deployment)
- Akeyless account with API key
- AWS S3 bucket with appropriate IAM permissions

## Configuration

The application is configured entirely through environment variables:

### Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `AKEYLESS_ACCESS_ID` | Akeyless Access ID for authentication | `p-xxxxxxxxxxxx` |
| `AKEYLESS_ACCESS_KEY` | Akeyless Access Key (secret) | `your-access-key-secret` |
| `AKEYLESS_GATEWAY_URL` | Akeyless gateway URL | `https://api.akeyless.io` |
| `AWS_REGION` | AWS region for S3 | `us-east-1` |
| `AWS_ACCESS_KEY_ID` | AWS access key ID | `AKIAIOSFODNN7EXAMPLE` |
| `AWS_SECRET_ACCESS_KEY` | AWS secret access key | `wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY` |
| `S3_BUCKET` | Target S3 bucket name | `my-secrets-backup` |

### Optional Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `BASE_PATH` | Base path in Akeyless to start from | `/` |
| `S3_ENDPOINT` | Custom S3 endpoint URL (for MinIO, DigitalOcean Spaces, etc.) | None (uses AWS S3) |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | `info` |

## Local Development

### Setup

1. Clone the repository:
```bash
git clone https://github.com/Jdavid77/akeylesstos3.git
cd akeylesstos3
```

2. Install dependencies:
```bash
go mod download
```

3. Create a `.env` file from the example:
```bash
cp .env.example .env
# Edit .env with your actual credentials
```

4. Run the application:
```bash
# Load environment variables
export $(cat .env | xargs)

# Run the application
go run main.go
```

### Build

```bash
go build -o akeylesstos3 .
```

## Docker

### Build Image

```bash
docker build -t akeylesstos3:latest .
```

### Run Container

```bash
docker run --rm \
  -e AKEYLESS_ACCESS_ID="your-access-id" \
  -e AKEYLESS_ACCESS_KEY="your-access-key" \
  -e AKEYLESS_GATEWAY_URL="https://api.akeyless.io" \
  -e AWS_REGION="us-east-1" \
  -e AWS_ACCESS_KEY_ID="your-aws-access-key-id" \
  -e AWS_SECRET_ACCESS_KEY="your-aws-secret-key" \
  -e S3_BUCKET="your-bucket" \
  -e S3_ENDPOINT="https://s3.example.com" \
  akeylesstos3:latest
```

**Note:** The `S3_ENDPOINT` is optional. Omit it for standard AWS S3. Use it for S3-compatible services like MinIO (`http://minio:9000`), DigitalOcean Spaces (`https://nyc3.digitaloceanspaces.com`), etc.

## Kubernetes Deployment

### Prerequisites

1. Build and push your Docker image to a container registry:
```bash
docker build -t your-registry/akeylesstos3:latest .
docker push your-registry/akeylesstos3:latest
```

2. Update the image in `k8s/cronjob.yaml` to point to your registry.

### Deploy

1. Create the secrets and configmaps:
```bash
# Copy the example and fill in your actual values
cp k8s/secrets-example.yaml k8s/secrets.yaml

# Edit k8s/secrets.yaml with your credentials
# Then apply it
kubectl apply -f k8s/secrets.yaml
```

2. Deploy the CronJob:
```bash
kubectl apply -f k8s/cronjob.yaml
```

3. Verify the deployment:
```bash
# Check CronJob status
kubectl get cronjob akeyless-to-s3-exporter

# View job history
kubectl get jobs --selector=app=akeyless-to-s3-exporter

# Check logs of the most recent job
kubectl logs -l app=akeyless-to-s3-exporter --tail=100
```

### Manual Trigger

To manually trigger the CronJob without waiting for the schedule:

```bash
kubectl create job --from=cronjob/akeyless-to-s3-exporter manual-run-$(date +%s)
```

## Output Format

Each secret is stored in S3 as a JSON file with the following structure:

```json
{
  "name": "database-password",
  "path": "/production/database/password",
  "value": "secret-value",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

The S3 key preserves the Akeyless folder structure. For example:
- Akeyless path: `/production/database/password`
- S3 key: `production/database/password.json`

## Logging

The application uses structured JSON logging. Example log output:

```json
{"level":"info","time":"2024-01-15T10:30:00Z","message":"Starting Akeyless to S3 exporter"}
{"level":"info","time":"2024-01-15T10:30:01Z","message":"Successfully authenticated with Akeyless"}
{"level":"info","base_path":"/","time":"2024-01-15T10:30:01Z","message":"Starting to list all secrets"}
{"level":"info","count":42,"time":"2024-01-15T10:30:05Z","message":"Found secrets"}
{"level":"info","secret_path":"/app/api-key","s3_key":"app/api-key.json","time":"2024-01-15T10:30:10Z","message":"Successfully uploaded secret"}
{"level":"info","total_secrets":42,"time":"2024-01-15T10:30:15Z","message":"Successfully exported all secrets to S3"}
```

## Error Handling

The application includes robust error handling:

- **Failed Authentication**: The application exits with a non-zero status code if authentication fails
- **Secret Retrieval Errors**: Failed secret retrievals are logged but don't stop the process; other secrets continue to be processed
- **Upload Failures**: Failed uploads are retried up to 3 times with exponential backoff
- **Final Status**: If any 49,90 €
￼￼
Como você gostaria de fazer o pagamento?
Cartões de crédito
Pagamento seguro com Visa e Mastercard
Alternative Payments with Checkout.com
MB WAY
Multi Banco
Country Code 
￼
Número de telefone 
￼uploads fail after all retries, the application exits with a non-zero status code

## Performance

- **Concurrency**: Up to 10 concurrent S3 uploads
- **Retry Logic**: 3 retries per upload with 2-second base delay
- **Memory**: Typical usage ~128MB, limit set to 512MB in Kubernetes
- **CPU**: Typical usage ~100m, limit set to 500m in Kubernetes

## Security Considerations

- API keys and AWS credentials are stored in Kubernetes Secrets
- The Docker container runs as a non-root user
- Secrets are transmitted over HTTPS
- Logs do not contain secret values
- For production use, consider enabling encryption at rest on your S3 bucket

## IAM Permissions

The AWS credentials need the following S3 permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:PutObject",
        "s3:PutObjectAcl"
      ],
      "Resource": "arn:aws:s3:::your-bucket-name/*"
    }
  ]
}
```

## Troubleshooting

### Application fails to start

Check that all required environment variables are set:
```bash
kubectl logs -l app=akeyless-to-s3-exporter
```

### Authentication failures

Verify your Akeyless API key and gateway URL:
```bash
kubectl get secret akeyless-credentials -o yaml
kubectl get configmap akeyless-config -o yaml
```

### S3 upload failures

Check AWS credentials and S3 bucket permissions:
```bash
kubectl get secret aws-credentials -o yaml
```

### Enable debug logging

Set `LOG_LEVEL=debug` in the environment variables for more detailed logs.

## License

MIT

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.
