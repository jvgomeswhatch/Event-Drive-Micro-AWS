locals {
  env  = var.environment
  proj = var.project
}

# ─── Lambda execution role ────────────────────────────────────────────────────
resource "aws_iam_role" "lambda_exec" {
  name = "${local.proj}-lambda-exec-${local.env}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect    = "Allow"
        Principal = { Service = "lambda.amazonaws.com" }
        Action    = "sts:AssumeRole"
      }
    ]
  })

  tags = merge(var.tags, { Name = "${local.proj}-lambda-exec-${local.env}" })
}

# ─── Lambda: CloudWatch Logs ──────────────────────────────────────────────────
resource "aws_iam_policy" "lambda_logs" {
  name        = "${local.proj}-lambda-logs-${local.env}"
  description = "Allows Lambda to write to CloudWatch Logs"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["logs:CreateLogGroup", "logs:CreateLogStream", "logs:PutLogEvents"]
        Resource = "arn:aws:logs:*:*:*"
      }
    ]
  })

  tags = var.tags
}

# ─── Lambda: SQS consumer ─────────────────────────────────────────────────────
resource "aws_iam_policy" "lambda_sqs" {
  name        = "${local.proj}-lambda-sqs-${local.env}"
  description = "Least-privilege SQS access for Lambda consumers"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "sqs:ReceiveMessage",
          "sqs:DeleteMessage",
          "sqs:GetQueueAttributes",
          "sqs:ChangeMessageVisibility"
        ]
        Resource = "arn:aws:sqs:*:*:*-queue-${local.env}"
      },
      {
        # Allow sending to DLQ on explicit requeue
        Effect   = "Allow"
        Action   = ["sqs:SendMessage"]
        Resource = "arn:aws:sqs:*:*:*-dlq-${local.env}"
      }
    ]
  })

  tags = var.tags
}

# ─── Lambda: SNS publisher ────────────────────────────────────────────────────
resource "aws_iam_policy" "lambda_sns" {
  name        = "${local.proj}-lambda-sns-${local.env}"
  description = "Allows Lambda to publish to SNS topics"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["sns:Publish"]
        Resource = "arn:aws:sns:*:*:*-${local.env}"
      }
    ]
  })

  tags = var.tags
}

# ─── Lambda: DynamoDB access ──────────────────────────────────────────────────
resource "aws_iam_policy" "lambda_dynamodb" {
  name        = "${local.proj}-lambda-dynamodb-${local.env}"
  description = "Least-privilege DynamoDB access for Lambda"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "dynamodb:PutItem",
          "dynamodb:GetItem",
          "dynamodb:UpdateItem",
          "dynamodb:DeleteItem",
          "dynamodb:Query",
          "dynamodb:ConditionCheckItem"
        ]
        Resource = [
          "arn:aws:dynamodb:*:*:table/*-${local.env}",
          "arn:aws:dynamodb:*:*:table/*-${local.env}/index/*"
        ]
      }
    ]
  })

  tags = var.tags
}

# ─── Attach all policies to Lambda role ───────────────────────────────────────
resource "aws_iam_role_policy_attachment" "lambda_logs" {
  role       = aws_iam_role.lambda_exec.name
  policy_arn = aws_iam_policy.lambda_logs.arn
}

resource "aws_iam_role_policy_attachment" "lambda_sqs" {
  role       = aws_iam_role.lambda_exec.name
  policy_arn = aws_iam_policy.lambda_sqs.arn
}

resource "aws_iam_role_policy_attachment" "lambda_sns" {
  role       = aws_iam_role.lambda_exec.name
  policy_arn = aws_iam_policy.lambda_sns.arn
}

resource "aws_iam_role_policy_attachment" "lambda_dynamodb" {
  role       = aws_iam_role.lambda_exec.name
  policy_arn = aws_iam_policy.lambda_dynamodb.arn
}

# ─── Service-specific roles (order, payment, inventory, notification) ─────────
# Each service gets its own role to enforce least-privilege at service level.
resource "aws_iam_role" "service" {
  for_each = toset(["order", "payment", "inventory", "notification"])

  name = "${local.proj}-${each.key}-svc-${local.env}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect    = "Allow"
        Principal = { Service = "lambda.amazonaws.com" }
        Action    = "sts:AssumeRole"
      }
    ]
  })

  tags = merge(var.tags, {
    Name    = "${local.proj}-${each.key}-svc-${local.env}"
    Service = each.key
  })
}

# ─── Service-scoped SQS policies (least-privilege per service) ────────────────
# Each service can only read from its own queue and write to its own DLQ.
# order-service additionally needs SendMessage to publish to the order queue.

resource "aws_iam_policy" "order_svc_sqs" {
  name        = "${local.proj}-order-svc-sqs-${local.env}"
  description = "order-service: send to order queue, read its own queue, write to DLQ"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["sqs:SendMessage"]
        Resource = "arn:aws:sqs:*:*:${local.proj}-order-queue-${local.env}"
      },
      {
        Effect   = "Allow"
        Action   = ["sqs:ReceiveMessage", "sqs:DeleteMessage", "sqs:GetQueueAttributes", "sqs:ChangeMessageVisibility"]
        Resource = "arn:aws:sqs:*:*:${local.proj}-order-queue-${local.env}"
      },
      {
        Effect   = "Allow"
        Action   = ["sqs:SendMessage"]
        Resource = "arn:aws:sqs:*:*:${local.proj}-order-dlq-${local.env}"
      }
    ]
  })

  tags = var.tags
}

resource "aws_iam_policy" "payment_svc_sqs" {
  name        = "${local.proj}-payment-svc-sqs-${local.env}"
  description = "payment-service: read payment queue, write to payment DLQ only"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["sqs:ReceiveMessage", "sqs:DeleteMessage", "sqs:GetQueueAttributes", "sqs:ChangeMessageVisibility"]
        Resource = "arn:aws:sqs:*:*:${local.proj}-payment-queue-${local.env}"
      },
      {
        Effect   = "Allow"
        Action   = ["sqs:SendMessage"]
        Resource = "arn:aws:sqs:*:*:${local.proj}-payment-dlq-${local.env}"
      }
    ]
  })

  tags = var.tags
}

resource "aws_iam_policy" "inventory_svc_sqs" {
  name        = "${local.proj}-inventory-svc-sqs-${local.env}"
  description = "inventory-service: read inventory queue, write to inventory DLQ only"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["sqs:ReceiveMessage", "sqs:DeleteMessage", "sqs:GetQueueAttributes", "sqs:ChangeMessageVisibility"]
        Resource = "arn:aws:sqs:*:*:${local.proj}-inventory-queue-${local.env}"
      },
      {
        Effect   = "Allow"
        Action   = ["sqs:SendMessage"]
        Resource = "arn:aws:sqs:*:*:${local.proj}-inventory-dlq-${local.env}"
      }
    ]
  })

  tags = var.tags
}

resource "aws_iam_policy" "notification_svc_sqs" {
  name        = "${local.proj}-notification-svc-sqs-${local.env}"
  description = "notification-service: read notification queue, write to notification DLQ only"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["sqs:ReceiveMessage", "sqs:DeleteMessage", "sqs:GetQueueAttributes", "sqs:ChangeMessageVisibility"]
        Resource = "arn:aws:sqs:*:*:${local.proj}-notification-queue-${local.env}"
      },
      {
        Effect   = "Allow"
        Action   = ["sqs:SendMessage"]
        Resource = "arn:aws:sqs:*:*:${local.proj}-notification-dlq-${local.env}"
      }
    ]
  })

  tags = var.tags
}

# ─── Service-scoped SNS policies ──────────────────────────────────────────────
# order-service publishes to order-events only.
# payment-service publishes to payment-events only.
# inventory-service publishes to inventory-events only.
# notification-service does not publish to SNS.

resource "aws_iam_policy" "order_svc_sns" {
  name        = "${local.proj}-order-svc-sns-${local.env}"
  description = "order-service: publish to order-events topic only"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["sns:Publish"]
      Resource = "arn:aws:sns:*:*:${local.proj}-order-events-${local.env}"
    }]
  })

  tags = var.tags
}

resource "aws_iam_policy" "payment_svc_sns" {
  name        = "${local.proj}-payment-svc-sns-${local.env}"
  description = "payment-service: publish to payment-events topic only"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["sns:Publish"]
      Resource = "arn:aws:sns:*:*:${local.proj}-payment-events-${local.env}"
    }]
  })

  tags = var.tags
}

resource "aws_iam_policy" "inventory_svc_sns" {
  name        = "${local.proj}-inventory-svc-sns-${local.env}"
  description = "inventory-service: publish to inventory-events topic only"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["sns:Publish"]
      Resource = "arn:aws:sns:*:*:${local.proj}-inventory-events-${local.env}"
    }]
  })

  tags = var.tags
}

# ─── Service-scoped DynamoDB policies ─────────────────────────────────────────
# Each service gets access only to the tables it legitimately needs.

resource "aws_iam_policy" "order_svc_dynamodb" {
  name        = "${local.proj}-order-svc-dynamodb-${local.env}"
  description = "order-service: orders table + idempotency table"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = ["dynamodb:PutItem", "dynamodb:GetItem", "dynamodb:UpdateItem", "dynamodb:Query", "dynamodb:ConditionCheckItem"]
      Resource = [
        "arn:aws:dynamodb:*:*:table/${local.proj}-orders-${local.env}",
        "arn:aws:dynamodb:*:*:table/${local.proj}-orders-${local.env}/index/*",
        "arn:aws:dynamodb:*:*:table/${local.proj}-idempotency-${local.env}",
      ]
    }]
  })

  tags = var.tags
}

resource "aws_iam_policy" "payment_svc_dynamodb" {
  name        = "${local.proj}-payment-svc-dynamodb-${local.env}"
  description = "payment-service: payments table + idempotency + event-timeline"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = ["dynamodb:PutItem", "dynamodb:GetItem", "dynamodb:UpdateItem", "dynamodb:Query", "dynamodb:ConditionCheckItem"]
      Resource = [
        "arn:aws:dynamodb:*:*:table/${local.proj}-payments-${local.env}",
        "arn:aws:dynamodb:*:*:table/${local.proj}-idempotency-${local.env}",
        "arn:aws:dynamodb:*:*:table/${local.proj}-event-timeline-${local.env}",
      ]
    }]
  })

  tags = var.tags
}

resource "aws_iam_policy" "inventory_svc_dynamodb" {
  name        = "${local.proj}-inventory-svc-dynamodb-${local.env}"
  description = "inventory-service: inventory table + idempotency + event-timeline"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = ["dynamodb:PutItem", "dynamodb:GetItem", "dynamodb:UpdateItem", "dynamodb:Query", "dynamodb:ConditionCheckItem"]
      Resource = [
        "arn:aws:dynamodb:*:*:table/${local.proj}-inventory-${local.env}",
        "arn:aws:dynamodb:*:*:table/${local.proj}-idempotency-${local.env}",
        "arn:aws:dynamodb:*:*:table/${local.proj}-event-timeline-${local.env}",
      ]
    }]
  })

  tags = var.tags
}

resource "aws_iam_policy" "notification_svc_dynamodb" {
  name        = "${local.proj}-notification-svc-dynamodb-${local.env}"
  description = "notification-service: orders (status update) + idempotency + event-timeline. Read-only on orders except UpdateItem."

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = ["dynamodb:UpdateItem", "dynamodb:PutItem", "dynamodb:ConditionCheckItem"]
      Resource = [
        "arn:aws:dynamodb:*:*:table/${local.proj}-orders-${local.env}",
        "arn:aws:dynamodb:*:*:table/${local.proj}-idempotency-${local.env}",
        "arn:aws:dynamodb:*:*:table/${local.proj}-event-timeline-${local.env}",
      ]
    }]
  })

  tags = var.tags
}

# ─── Attach service-scoped policies to service roles ──────────────────────────

resource "aws_iam_role_policy_attachment" "order_svc_sqs" {
  role       = aws_iam_role.service["order"].name
  policy_arn = aws_iam_policy.order_svc_sqs.arn
}

resource "aws_iam_role_policy_attachment" "order_svc_sns" {
  role       = aws_iam_role.service["order"].name
  policy_arn = aws_iam_policy.order_svc_sns.arn
}

resource "aws_iam_role_policy_attachment" "order_svc_dynamodb" {
  role       = aws_iam_role.service["order"].name
  policy_arn = aws_iam_policy.order_svc_dynamodb.arn
}

resource "aws_iam_role_policy_attachment" "payment_svc_sqs" {
  role       = aws_iam_role.service["payment"].name
  policy_arn = aws_iam_policy.payment_svc_sqs.arn
}

resource "aws_iam_role_policy_attachment" "payment_svc_sns" {
  role       = aws_iam_role.service["payment"].name
  policy_arn = aws_iam_policy.payment_svc_sns.arn
}

resource "aws_iam_role_policy_attachment" "payment_svc_dynamodb" {
  role       = aws_iam_role.service["payment"].name
  policy_arn = aws_iam_policy.payment_svc_dynamodb.arn
}

resource "aws_iam_role_policy_attachment" "inventory_svc_sqs" {
  role       = aws_iam_role.service["inventory"].name
  policy_arn = aws_iam_policy.inventory_svc_sqs.arn
}

resource "aws_iam_role_policy_attachment" "inventory_svc_sns" {
  role       = aws_iam_role.service["inventory"].name
  policy_arn = aws_iam_policy.inventory_svc_sns.arn
}

resource "aws_iam_role_policy_attachment" "inventory_svc_dynamodb" {
  role       = aws_iam_role.service["inventory"].name
  policy_arn = aws_iam_policy.inventory_svc_dynamodb.arn
}

resource "aws_iam_role_policy_attachment" "notification_svc_sqs" {
  role       = aws_iam_role.service["notification"].name
  policy_arn = aws_iam_policy.notification_svc_sqs.arn
}

resource "aws_iam_role_policy_attachment" "notification_svc_dynamodb" {
  role       = aws_iam_role.service["notification"].name
  policy_arn = aws_iam_policy.notification_svc_dynamodb.arn
}
