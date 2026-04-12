variable "region" {
  default = "us-east-1"
}

variable "opensearch_domain_endpoint" {
  description = "OpenSearch domain endpoint (https://search-xxx.region.es.amazonaws.com)"
  type        = string
}

variable "proxy_image" {
  default = "ghcr.io/seraphjiang/oauth4os:latest"
}

variable "proxy_cpu" {
  default = 256
}

variable "proxy_memory" {
  default = 512
}

variable "desired_count" {
  default = 2
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

# --- Networking ---

data "aws_vpc" "default" {
  default = true
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

resource "aws_security_group" "proxy" {
  name_prefix = "oauth4os-"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# --- ECS ---

resource "aws_ecs_cluster" "proxy" {
  name = "oauth4os"
}

resource "aws_cloudwatch_log_group" "proxy" {
  name              = "/ecs/oauth4os"
  retention_in_days = 14
}

resource "aws_iam_role" "task_execution" {
  name = "oauth4os-task-execution"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action    = "sts:AssumeRole"
      Effect    = "Allow"
      Principal = { Service = "ecs-tasks.amazonaws.com" }
    }]
  })
}

resource "aws_iam_role_policy_attachment" "task_execution" {
  role       = aws_iam_role.task_execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

resource "aws_ecs_task_definition" "proxy" {
  family                   = "oauth4os"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = var.proxy_cpu
  memory                   = var.proxy_memory
  execution_role_arn       = aws_iam_role.task_execution.arn

  container_definitions = jsonencode([{
    name      = "oauth4os"
    image     = var.proxy_image
    essential = true
    portMappings = [{ containerPort = 8443, protocol = "tcp" }]
    environment = [
      { name = "UPSTREAM_ENGINE", value = var.opensearch_domain_endpoint },
      { name = "UPSTREAM_DASHBOARDS", value = "${var.opensearch_domain_endpoint}/_dashboards" },
    ]
    logConfiguration = {
      logDriver = "awslogs"
      options = {
        "awslogs-group"         = aws_cloudwatch_log_group.proxy.name
        "awslogs-region"        = var.region
        "awslogs-stream-prefix" = "proxy"
      }
    }
  }])
}

resource "aws_ecs_service" "proxy" {
  name            = "oauth4os"
  cluster         = aws_ecs_cluster.proxy.id
  task_definition = aws_ecs_task_definition.proxy.arn
  desired_count   = var.desired_count
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = data.aws_subnets.default.ids
    security_groups  = [aws_security_group.proxy.id]
    assign_public_ip = true
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.proxy.arn
    container_name   = "oauth4os"
    container_port   = 8443
  }

  depends_on = [aws_lb_listener.proxy]
}

# --- ALB ---

resource "aws_lb" "proxy" {
  name               = "oauth4os"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.proxy.id]
  subnets            = data.aws_subnets.default.ids
}

resource "aws_lb_target_group" "proxy" {
  name        = "oauth4os"
  port        = 8443
  protocol    = "HTTP"
  vpc_id      = data.aws_vpc.default.id
  target_type = "ip"

  health_check {
    path                = "/health"
    interval            = 15
    healthy_threshold   = 2
    unhealthy_threshold = 3
  }
}

resource "aws_lb_listener" "proxy" {
  load_balancer_arn = aws_lb.proxy.arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.proxy.arn
  }
}

# --- Outputs ---

output "proxy_url" {
  value = "http://${aws_lb.proxy.dns_name}"
}

output "ecs_cluster" {
  value = aws_ecs_cluster.proxy.name
}

output "log_group" {
  value = aws_cloudwatch_log_group.proxy.name
}
