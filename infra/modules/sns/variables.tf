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

variable "topics" {
  description = "List of SNS topic base names to create"
  type        = list(string)
}

variable "subscriptions" {
  description = "SNS → SQS subscription definitions"
  type = list(object({
    topic  = string
    queue  = string
    filter = map(list(string))
  }))
  default = []
}

variable "sqs_queue_arns" {
  description = "Map of queue name → ARN from the SQS module"
  type        = map(string)
}
