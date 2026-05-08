variable "environment" {
  description = "Deployment environment"
  type        = string
  default     = "staging"

  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment must be one of: dev, staging, prod."
  }
}

variable "project_name" {
  description = "Project name used in resource naming"
  type        = string
  default     = "platform"
}

variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "localstack_endpoint" {
  description = "LocalStack endpoint URL"
  type        = string
  default     = "http://localstack:4566"
}

variable "order_service_port" {
  description = "Port where order-service listens"
  type        = number
  default     = 3001
}
