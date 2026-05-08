locals {
  topic_map = {
    for t in var.topics : t => "${t}-${var.environment}"
  }

  # Index subscriptions by position for unique keys
  subscriptions_indexed = {
    for idx, sub in var.subscriptions :
    "${sub.topic}-to-${sub.queue}-${idx}" => sub
  }
}

# ─── SNS Topics ───────────────────────────────────────────────────────────────
resource "aws_sns_topic" "this" {
  for_each = local.topic_map

  name = each.value

  tags = merge(var.tags, {
    Name  = each.value
    Topic = each.key
  })
}

# ─── SNS → SQS Subscriptions ──────────────────────────────────────────────────
resource "aws_sns_topic_subscription" "sqs" {
  for_each = local.subscriptions_indexed

  topic_arn = aws_sns_topic.this[each.value.topic].arn
  protocol  = "sqs"
  endpoint  = var.sqs_queue_arns["${each.value.queue}"]

  # Message attribute filter — only deliver matching events
  filter_policy = length(each.value.filter) > 0 ? jsonencode(each.value.filter) : null

  # Deliver the raw message body (no SNS envelope wrapper)
  raw_message_delivery = true
}
