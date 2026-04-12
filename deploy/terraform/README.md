# oauth4os Terraform Module

Deploy oauth4os to AWS AppRunner with ECR and IAM.

## Usage

```hcl
module "oauth4os" {
  source    = "./deploy/terraform"
  name      = "oauth4os"
  region    = "us-west-2"
  image_tag = "v1.0.1"
  cpu       = "256"
  memory    = "512"
}

output "url" { value = module.oauth4os.service_url }
```

## Deploy

```bash
cd deploy/terraform
terraform init
terraform plan
terraform apply
```

## Resources Created

- ECR repository with lifecycle policy (keep 10 images)
- IAM roles for AppRunner (ECR access + instance role)
- AppRunner service with auto-scaling (1-3 instances)
- Health check on `/health`
- Auto-deploy on ECR image push

## Inputs

| Name | Description | Default |
|------|-------------|---------|
| name | Resource name prefix | oauth4os |
| region | AWS region | us-west-2 |
| image_tag | Docker image tag | latest |
| cpu | AppRunner CPU | 256 |
| memory | AppRunner memory | 512 |

## Outputs

| Name | Description |
|------|-------------|
| service_url | AppRunner HTTPS URL |
| ecr_repository_url | ECR push target |
| service_arn | For CLI operations |
