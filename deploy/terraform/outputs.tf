output "service_url" {
  description = "AppRunner service URL"
  value       = "https://${aws_apprunner_service.this.service_url}"
}

output "ecr_repository_url" {
  description = "ECR repository URL"
  value       = aws_ecr_repository.this.repository_url
}

output "service_arn" {
  description = "AppRunner service ARN"
  value       = aws_apprunner_service.this.arn
}
