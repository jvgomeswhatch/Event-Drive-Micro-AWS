variable "environment" {
  type = string
}

variable "project" {
  type = string
}

variable "tags" {
  type    = map(string)
  default = {}
}

variable "lambda_role_arn" {
  description = "IAM role ARN for Lambda execution"
  type        = string
}

variable "order_queue_arn" {
  description = "SQS ARN for the order queue (event source mapping)"
  type        = string
}

variable "localstack_endpoint" {
  description = "LocalStack endpoint for Lambda env vars"
  type        = string
  default     = "http://localstack:4566"
}
