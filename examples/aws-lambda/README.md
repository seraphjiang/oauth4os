# oauth4os on AWS Lambda (Function URL)

Lightweight deployment — run oauth4os as a single Lambda function with a Function URL. No ALB, no ECS, no containers to manage.

## Architecture

```
Client → Lambda Function URL (:443) → oauth4os (Go binary) → OpenSearch Domain
```

## How It Works

Go compiles to a single binary. Lambda runs it via the `provided.al2023` runtime. The Function URL gives you a public HTTPS endpoint with no API Gateway needed.

## Deploy

```bash
cd examples/aws-lambda

# Build the Lambda binary
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bootstrap ../../cmd/proxy
zip function.zip bootstrap config.yaml

# Deploy
terraform init
terraform apply -var="opensearch_endpoint=https://search-xxx.us-east-1.es.amazonaws.com"
```

Or build in Docker (no local Go needed):

```bash
docker run --rm -v $(pwd)/../..:/app -w /app \
  golang:1.22 sh -c \
  'GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o examples/aws-lambda/bootstrap ./cmd/proxy'
cd examples/aws-lambda && zip function.zip bootstrap config.yaml
terraform apply -var="opensearch_endpoint=https://..."
```

## Variables

| Variable | Description | Default |
|---|---|---|
| `region` | AWS region | `us-east-1` |
| `opensearch_endpoint` | OpenSearch domain URL | (required) |
| `memory_size` | Lambda memory (MB) | `256` |
| `timeout` | Lambda timeout (seconds) | `30` |

## Outputs

| Output | Description |
|---|---|
| `function_url` | Public HTTPS endpoint |
| `function_name` | Lambda function name |

## Cost

~$0.50/month for 100K requests. No idle cost — true pay-per-request.

## Limitations

- Cold start: ~200-400ms (Go on ARM64 is fast)
- 15-minute max timeout per request
- No WebSocket support (use ECS Fargate for Dashboards proxy)
- Function URL has no WAF integration (use CloudFront + Function URL for WAF)
