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

variable "order_service_url" {
  description = "Base URL of the order-service (HTTP proxy target)"
  type        = string
}
