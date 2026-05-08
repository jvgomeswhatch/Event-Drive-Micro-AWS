locals {
  env  = var.environment
  proj = var.project

  # Each service has its own Lambda function definition
  functions = {
    order-processor = {
      description = "Processes incoming orders from API Gateway"
      handler     = "handler.handler"
      timeout     = 30
      memory_size = 256
      environment = {
        SERVICE_NAME        = "order-service"
        LOCALSTACK_ENDPOINT = var.localstack_endpoint
      }
    }
    payment-processor = {
      description = "Processes payment events from SQS"
      handler     = "handler.handler"
      timeout     = 60
      memory_size = 256
      environment = {
        SERVICE_NAME        = "payment-service"
        LOCALSTACK_ENDPOINT = var.localstack_endpoint
      }
    }
    inventory-processor = {
      description = "Processes inventory reservation from SQS"
      handler     = "handler.handler"
      timeout     = 30
      memory_size = 128
      environment = {
        SERVICE_NAME        = "inventory-service"
        LOCALSTACK_ENDPOINT = var.localstack_endpoint
      }
    }
    notification-processor = {
      description = "Aggregates events and sends notifications"
      handler     = "handler.handler"
      timeout     = 30
      memory_size = 128
      environment = {
        SERVICE_NAME        = "notification-service"
        LOCALSTACK_ENDPOINT = var.localstack_endpoint
      }
    }
  }
}

# ─── Lambda functions ─────────────────────────────────────────────────────────
# In LocalStack, we use a zip with a stub handler.
# In real AWS, these would be container-based (image_uri).
data "archive_file" "stub" {
  for_each = local.functions

  type        = "zip"
  output_path = "/tmp/${each.key}-stub.zip"

  source {
    content  = <<-EOF
      exports.handler = async (event) => {
        console.log(JSON.stringify({ service: '${each.key}', event }));
        return { statusCode: 200, body: 'stub' };
      };
    EOF
    filename = "handler.js"
  }
}

resource "aws_lambda_function" "this" {
  for_each = local.functions

  function_name = "${local.proj}-${each.key}-${local.env}"
  description   = each.value.description
  role          = var.lambda_role_arn
  handler       = each.value.handler
  runtime       = "nodejs20.x"
  timeout       = each.value.timeout
  memory_size   = each.value.memory_size

  filename         = data.archive_file.stub[each.key].output_path
  source_code_hash = data.archive_file.stub[each.key].output_base64sha256

  environment {
    variables = merge(each.value.environment, {
      ENVIRONMENT = local.env
    })
  }

  tags = merge(var.tags, {
    Name     = "${local.proj}-${each.key}-${local.env}"
    Function = each.key
  })
}

# ─── SQS event source mapping for order-processor ────────────────────────────
resource "aws_lambda_event_source_mapping" "order_queue" {
  event_source_arn = var.order_queue_arn
  function_name    = aws_lambda_function.this["order-processor"].arn
  batch_size       = 10
  enabled          = true

  function_response_types = ["ReportBatchItemFailures"]
}
