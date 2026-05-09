terraform {
  required_version = ">= 1.8"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    archive = {
      source  = "hashicorp/archive"
      version = "~> 2.0"
    }
  }

  # Remote state simulado via S3 (LocalStack)
  # Descomente após o primeiro apply manual para habilitar state remoto
  # backend "s3" {
  #   bucket                      = "terraform-state-dev"
  #   key                         = "platform/dev/terraform.tfstate"
  #   region                      = "us-east-1"
  #   endpoint                    = "http://localstack:4566"
  #   access_key                  = "test"
  #   secret_key                  = "test"
  #   skip_credentials_validation = true
  #   skip_metadata_api_check     = true
  #   skip_region_validation      = true
  #   force_path_style            = true
  # }
}

provider "aws" {
  region                      = var.aws_region
  access_key                  = "test"
  secret_key                  = "test"
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true

  endpoints {
    sqs        = var.localstack_endpoint
    sns        = var.localstack_endpoint
    dynamodb   = var.localstack_endpoint
    lambda     = var.localstack_endpoint
    apigateway = var.localstack_endpoint
    iam        = var.localstack_endpoint
    s3         = var.localstack_endpoint
    cloudwatch = var.localstack_endpoint
    logs       = var.localstack_endpoint
    sts        = var.localstack_endpoint
  }
}

locals {
  common_tags = {
    Environment = var.environment
    Project     = var.project_name
    ManagedBy   = "terraform"
    Owner       = "platform-team"
  }
}

# ─── IAM ──────────────────────────────────────────────────────────────────────
module "iam" {
  source      = "../../modules/iam"
  environment = var.environment
  project     = var.project_name
  tags        = local.common_tags
}

# ─── SQS ──────────────────────────────────────────────────────────────────────
module "sqs" {
  source      = "../../modules/sqs"
  environment = var.environment
  project     = var.project_name
  tags        = local.common_tags

  services = ["order", "payment", "inventory", "notification"]

  visibility_timeout_seconds = 30
  message_retention_seconds  = 86400
  max_receive_count          = 3
}

# ─── SNS ──────────────────────────────────────────────────────────────────────
module "sns" {
  source      = "../../modules/sns"
  environment = var.environment
  project     = var.project_name
  tags        = local.common_tags

  topics = ["order-events", "payment-events", "inventory-events"]

  # LocalStack 3.x não aplica filter_policy com confiança quando raw_message_delivery=true.
  # As filas de destino que têm consumers que já filtram internamente (notification-queue)
  # recebem tudo sem filtro SNS; a filtragem é feita no código do consumer.
  # Filas de processamento crítico (payment-queue, inventory-queue) mantêm o filtro.
  subscriptions = [
    {
      topic  = "order-events"
      queue  = "payment-queue"
      filter = { eventType = ["order.created", "order.updated", "order.cancelled"] }
    },
    {
      topic  = "payment-events"
      queue  = "inventory-queue"
      filter = { eventType = ["payment.succeeded"] }
    },
    {
      topic  = "payment-events"
      queue  = "notification-queue"
      filter = {}
    },
    {
      topic  = "inventory-events"
      queue  = "notification-queue"
      filter = {}
    },
  ]

  sqs_queue_arns = module.sqs.queue_arns
}

# ─── DynamoDB ─────────────────────────────────────────────────────────────────
module "dynamodb" {
  source      = "../../modules/dynamodb"
  environment = var.environment
  project     = var.project_name
  tags        = local.common_tags
}

# ─── Lambda ───────────────────────────────────────────────────────────────────
module "lambda" {
  source      = "../../modules/lambda"
  environment = var.environment
  project     = var.project_name
  tags        = local.common_tags

  lambda_role_arn     = module.iam.lambda_role_arn
  order_queue_arn     = module.sqs.queue_arns["order-queue"]
  localstack_endpoint = var.localstack_endpoint
}

# ─── API Gateway ──────────────────────────────────────────────────────────────
module "api_gateway" {
  source      = "../../modules/api_gateway"
  environment = var.environment
  project     = var.project_name
  tags        = local.common_tags

  order_service_url = "http://order-service:${var.order_service_port}"
}
