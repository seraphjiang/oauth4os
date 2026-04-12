variable "name" {
  description = "Name prefix for all resources"
  type        = string
  default     = "oauth4os"
}

variable "region" {
  description = "AWS region"
  type        = string
  default     = "us-west-2"
}

variable "image_tag" {
  description = "Docker image tag"
  type        = string
  default     = "latest"
}

variable "cpu" {
  description = "AppRunner CPU (256, 512, 1024, 2048, 4096)"
  type        = string
  default     = "256"
}

variable "memory" {
  description = "AppRunner memory (512, 1024, 2048, 3072, 4096, ...)"
  type        = string
  default     = "512"
}

variable "config_yaml" {
  description = "Path to config.yaml"
  type        = string
  default     = "config.yaml"
}

variable "tags" {
  description = "Tags for all resources"
  type        = map(string)
  default     = {}
}
