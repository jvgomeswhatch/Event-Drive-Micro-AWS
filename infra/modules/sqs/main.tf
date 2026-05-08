locals {
  # Build a map of service → { queue, dlq } for each service
  service_queues = {
    for svc in var.services : svc => {
      queue_name = "${svc}-queue-${var.environment}"
      dlq_name   = "${svc}-dlq-${var.environment}"
    }
  }
}

# ─── Dead Letter Queues (created first — referenced by main queues) ────────────
resource "aws_sqs_queue" "dlq" {
  for_each = local.service_queues

  name                       = each.value.dlq_name
  message_retention_seconds  = var.dlq_message_retention_seconds
  receive_wait_time_seconds  = 0

  tags = merge(var.tags, {
    Name    = each.value.dlq_name
    Service = each.key
    Type    = "dlq"
  })
}

# ─── Main Queues ──────────────────────────────────────────────────────────────
resource "aws_sqs_queue" "main" {
  for_each = local.service_queues

  name                       = each.value.queue_name
  visibility_timeout_seconds = var.visibility_timeout_seconds
  message_retention_seconds  = var.message_retention_seconds
  receive_wait_time_seconds  = 20 # long polling

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.dlq[each.key].arn
    maxReceiveCount     = var.max_receive_count
  })

  tags = merge(var.tags, {
    Name    = each.value.queue_name
    Service = each.key
    Type    = "main"
  })
}

# ─── Queue policies (allow SNS to publish) ────────────────────────────────────
resource "aws_sqs_queue_policy" "allow_sns" {
  for_each  = local.service_queues
  queue_url = aws_sqs_queue.main[each.key].id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AllowSNSPublish"
        Effect    = "Allow"
        Principal = { Service = "sns.amazonaws.com" }
        Action    = "sqs:SendMessage"
        Resource  = aws_sqs_queue.main[each.key].arn
      }
    ]
  })
}
