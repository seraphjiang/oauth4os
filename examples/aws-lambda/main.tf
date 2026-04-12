variable "region" {
  default = "us-east-1"
}

variable "opensearch_endpoint" {
  description = "OpenSearch domain endpoint (https://search-xxx.region.es.amazonaws.com)"
  type        = string
}

variable "memory_size" {
  default = 256
}

variable "timeout" {
  default = 30
}

terraform {
  required_version = ">= 1.5"
  required_providers {
    aws = { source = "hashicorp/aws", version = "~> 5.0" }
  }
}

provider "aws" {
  region = var.region
}

# --- IAM ---

resource "aws_iam_role" "lambda" {
  name = "oauth4os-lambda"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action    = "sts:AssumeRole"
      Effect    = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
    }]
  })
}

resource "aws_iam_role_policy_attachment" "lambda_basic" {
  role       = aws_iam_role.lambda.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

# --- Lambda ---

resource "aws_lambda_function" "proxy" {
  function_name = "oauth4os"
  role          = aws_iam_role.lambda.arn
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]
  memory_size   = var.memory_size
  timeout       = var.timeout
  filename      = "function.zip"

  environment {
    variables = {
      UPSTREAM_ENGINE     = var.opensearch_endpoint
      UPSTREAM_DASHBOARDS = "${var.opensearch_endpoint}/_dashboards"
      LISTEN              = ":8080"
    }
  }
}

resource "aws_lambda_function_url" "proxy" {
  function_name      = aws_lambda_function.proxy.function_name
  authorization_type = "NONE"
}

resource "aws_cloudwatch_log_group" "proxy" {
  name              = "/aws/lambda/oauth4os"
  retention_in_days = 14
}

# --- Outputs ---

output "function_url" {
  value = aws_lambda_function_url.proxy.function_url
}

output "function_name" {
  value = aws_lambda_function.proxy.function_name
}
