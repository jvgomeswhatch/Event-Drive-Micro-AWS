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

variable "services" {
  description = "List of service names to create queues for"
  type        = list(string)
}

variable "visibility_timeout_seconds" {
  description = "How long a message is hidden after being received"
  type        = number
  default     = 30
}

variable "message_retention_seconds" {
  description = "How long SQS retains a message"
  type        = number
  default     = 86400 # 24h
}

variable "dlq_message_retention_seconds" {
  description = "How long the DLQ retains a failed message"
  type        = number
  default     = 1209600 # 14 days
}

variable "max_receive_count" {
  description = "Max delivery attempts before routing to DLQ"
  type        = number
  default     = 3
}
