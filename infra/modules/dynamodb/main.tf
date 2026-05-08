locals {
  env = var.environment
}

# ─── Orders table ─────────────────────────────────────────────────────────────
resource "aws_dynamodb_table" "orders" {
  name         = "orders-${local.env}"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "orderId"

  attribute {
    name = "orderId"
    type = "S"
  }

  attribute {
    name = "customerId"
    type = "S"
  }

  attribute {
    name = "status"
    type = "S"
  }

  attribute {
    name = "createdAt"
    type = "S"
  }

  # GSI: query all orders for a customer
  global_secondary_index {
    name            = "customerId-createdAt-index"
    hash_key        = "customerId"
    range_key       = "createdAt"
    projection_type = "ALL"
  }

  # GSI: query orders by status (admin/ops view)
  global_secondary_index {
    name            = "status-createdAt-index"
    hash_key        = "status"
    range_key       = "createdAt"
    projection_type = "ALL"
  }

  ttl {
    attribute_name = "expiresAt"
    enabled        = true
  }

  tags = merge(var.tags, { Name = "orders-${local.env}" })
}

# ─── Payments table ───────────────────────────────────────────────────────────
resource "aws_dynamodb_table" "payments" {
  name         = "payments-${local.env}"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "paymentId"

  attribute {
    name = "paymentId"
    type = "S"
  }

  attribute {
    name = "orderId"
    type = "S"
  }

  # GSI: look up payment by orderId
  global_secondary_index {
    name            = "orderId-index"
    hash_key        = "orderId"
    projection_type = "ALL"
  }

  tags = merge(var.tags, { Name = "payments-${local.env}" })
}

# ─── Inventory table ──────────────────────────────────────────────────────────
resource "aws_dynamodb_table" "inventory" {
  name         = "inventory-${local.env}"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "productId"

  attribute {
    name = "productId"
    type = "S"
  }

  tags = merge(var.tags, { Name = "inventory-${local.env}" })
}

# ─── Idempotency table ────────────────────────────────────────────────────────
# Prevents duplicate event processing across all services.
# Key: {serviceId}#{messageId}, TTL expires after 24h.
resource "aws_dynamodb_table" "idempotency" {
  name         = "idempotency-${local.env}"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "idempotencyKey"

  attribute {
    name = "idempotencyKey"
    type = "S"
  }

  ttl {
    attribute_name = "expiresAt"
    enabled        = true
  }

  tags = merge(var.tags, { Name = "idempotency-${local.env}" })
}

# ─── Event timeline table ─────────────────────────────────────────────────────
# Stores per-order event history for the frontend timeline view.
resource "aws_dynamodb_table" "event_timeline" {
  name         = "event-timeline-${local.env}"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "orderId"
  range_key    = "eventId"

  attribute {
    name = "orderId"
    type = "S"
  }

  attribute {
    name = "eventId"
    type = "S"
  }

  ttl {
    attribute_name = "expiresAt"
    enabled        = true
  }

  tags = merge(var.tags, { Name = "event-timeline-${local.env}" })
}
