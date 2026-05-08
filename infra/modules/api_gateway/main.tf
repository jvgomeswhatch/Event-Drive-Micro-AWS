locals {
  env      = var.environment
  proj     = var.project
  api_name = "${local.proj}-api-${local.env}"
}

# ─── REST API ─────────────────────────────────────────────────────────────────
resource "aws_api_gateway_rest_api" "this" {
  name        = local.api_name
  description = "Platform REST API — ${local.env}"

  endpoint_configuration {
    types = ["REGIONAL"]
  }

  tags = merge(var.tags, { Name = local.api_name })
}

# ─── /orders resource ─────────────────────────────────────────────────────────
resource "aws_api_gateway_resource" "orders" {
  rest_api_id = aws_api_gateway_rest_api.this.id
  parent_id   = aws_api_gateway_rest_api.this.root_resource_id
  path_part   = "orders"
}

resource "aws_api_gateway_resource" "order_id" {
  rest_api_id = aws_api_gateway_rest_api.this.id
  parent_id   = aws_api_gateway_resource.orders.id
  path_part   = "{orderId}"
}

# ─── POST /orders ──────────────────────────────────────────────────────────────
resource "aws_api_gateway_method" "create_order" {
  rest_api_id      = aws_api_gateway_rest_api.this.id
  resource_id      = aws_api_gateway_resource.orders.id
  http_method      = "POST"
  authorization    = "NONE"
  api_key_required = false

  request_validator_id = aws_api_gateway_request_validator.body.id

  request_models = {
    "application/json" = aws_api_gateway_model.create_order.name
  }
}

resource "aws_api_gateway_integration" "create_order" {
  rest_api_id             = aws_api_gateway_rest_api.this.id
  resource_id             = aws_api_gateway_resource.orders.id
  http_method             = aws_api_gateway_method.create_order.http_method
  type                    = "HTTP_PROXY"
  integration_http_method = "POST"
  uri                     = "${var.order_service_url}/orders"

  request_templates = {
    "application/json" = "$input.body"
  }
}

# ─── GET /orders ───────────────────────────────────────────────────────────────
resource "aws_api_gateway_method" "list_orders" {
  rest_api_id   = aws_api_gateway_rest_api.this.id
  resource_id   = aws_api_gateway_resource.orders.id
  http_method   = "GET"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "list_orders" {
  rest_api_id             = aws_api_gateway_rest_api.this.id
  resource_id             = aws_api_gateway_resource.orders.id
  http_method             = aws_api_gateway_method.list_orders.http_method
  type                    = "HTTP_PROXY"
  integration_http_method = "GET"
  uri                     = "${var.order_service_url}/orders"
}

# ─── GET /orders/{orderId} ─────────────────────────────────────────────────────
resource "aws_api_gateway_method" "get_order" {
  rest_api_id   = aws_api_gateway_rest_api.this.id
  resource_id   = aws_api_gateway_resource.order_id.id
  http_method   = "GET"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "get_order" {
  rest_api_id             = aws_api_gateway_rest_api.this.id
  resource_id             = aws_api_gateway_resource.order_id.id
  http_method             = aws_api_gateway_method.get_order.http_method
  type                    = "HTTP_PROXY"
  integration_http_method = "GET"
  uri                     = "${var.order_service_url}/orders/{orderId}"

  request_parameters = {
    "integration.request.path.orderId" = "method.request.path.orderId"
  }
}

# ─── Request validator ────────────────────────────────────────────────────────
resource "aws_api_gateway_request_validator" "body" {
  name                        = "body-validator"
  rest_api_id                 = aws_api_gateway_rest_api.this.id
  validate_request_body       = true
  validate_request_parameters = false
}

# ─── Request model for POST /orders ───────────────────────────────────────────
resource "aws_api_gateway_model" "create_order" {
  rest_api_id  = aws_api_gateway_rest_api.this.id
  name         = "CreateOrderRequest"
  content_type = "application/json"

  schema = jsonencode({
    "$schema" = "http://json-schema.org/draft-04/schema#"
    type      = "object"
    required  = ["customerId", "items"]
    properties = {
      customerId = {
        type      = "string"
        minLength = 1
      }
      items = {
        type     = "array"
        minItems = 1
        items = {
          type     = "object"
          required = ["productId", "quantity"]
          properties = {
            productId = { type = "string" }
            quantity  = { type = "integer", minimum = 1 }
          }
        }
      }
      simulateFailure = {
        type = "boolean"
      }
    }
  })
}

# ─── CORS (OPTIONS method) ────────────────────────────────────────────────────
resource "aws_api_gateway_method" "options_orders" {
  rest_api_id   = aws_api_gateway_rest_api.this.id
  resource_id   = aws_api_gateway_resource.orders.id
  http_method   = "OPTIONS"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "options_orders" {
  rest_api_id = aws_api_gateway_rest_api.this.id
  resource_id = aws_api_gateway_resource.orders.id
  http_method = aws_api_gateway_method.options_orders.http_method
  type        = "MOCK"

  request_templates = {
    "application/json" = "{\"statusCode\": 200}"
  }
}

resource "aws_api_gateway_method_response" "options_orders" {
  rest_api_id = aws_api_gateway_rest_api.this.id
  resource_id = aws_api_gateway_resource.orders.id
  http_method = aws_api_gateway_method.options_orders.http_method
  status_code = "200"

  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = true
    "method.response.header.Access-Control-Allow-Methods" = true
    "method.response.header.Access-Control-Allow-Origin"  = true
  }
}

resource "aws_api_gateway_integration_response" "options_orders" {
  rest_api_id = aws_api_gateway_rest_api.this.id
  resource_id = aws_api_gateway_resource.orders.id
  http_method = aws_api_gateway_method.options_orders.http_method
  status_code = aws_api_gateway_method_response.options_orders.status_code

  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = "'Content-Type,Authorization,X-Correlation-ID,X-Idempotency-Key'"
    "method.response.header.Access-Control-Allow-Methods" = "'GET,POST,OPTIONS'"
    "method.response.header.Access-Control-Allow-Origin"  = "'*'"
  }
}

# ─── Deployment + Stage ───────────────────────────────────────────────────────
resource "aws_api_gateway_deployment" "this" {
  rest_api_id = aws_api_gateway_rest_api.this.id

  depends_on = [
    aws_api_gateway_integration.create_order,
    aws_api_gateway_integration.list_orders,
    aws_api_gateway_integration.get_order,
    aws_api_gateway_integration.options_orders,
  ]

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_api_gateway_stage" "this" {
  deployment_id = aws_api_gateway_deployment.this.id
  rest_api_id   = aws_api_gateway_rest_api.this.id
  stage_name    = local.env

  tags = merge(var.tags, { Name = "${local.api_name}-stage-${local.env}" })
}
