# oauth4os on ECS Fargate + Amazon OpenSearch

Deploy oauth4os proxy in front of an Amazon OpenSearch domain using ECS Fargate.

## Architecture

```
Client → ALB (:443) → ECS Fargate (oauth4os :8443) → OpenSearch Domain (:443)
```

## Usage

```bash
cd examples/terraform
terraform init
terraform plan -var="opensearch_domain_endpoint=https://search-xxx.us-east-1.es.amazonaws.com"
terraform apply
```

## Variables

| Variable | Description | Default |
|---|---|---|
| `region` | AWS region | `us-east-1` |
| `opensearch_domain_endpoint` | OpenSearch domain URL | (required) |
| `proxy_image` | oauth4os Docker image | `ghcr.io/seraphjiang/oauth4os:latest` |
| `proxy_cpu` | Fargate CPU units | `256` |
| `proxy_memory` | Fargate memory (MB) | `512` |
| `desired_count` | Number of proxy tasks | `2` |

## Outputs

| Output | Description |
|---|---|
| `proxy_url` | ALB URL for the oauth4os proxy |
| `ecs_cluster` | ECS cluster name |
| `log_group` | CloudWatch log group |
