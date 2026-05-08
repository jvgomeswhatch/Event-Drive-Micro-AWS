output "table_names" {
  description = "Map of logical name → DynamoDB table name"
  value = {
    orders         = aws_dynamodb_table.orders.name
    payments       = aws_dynamodb_table.payments.name
    inventory      = aws_dynamodb_table.inventory.name
    idempotency    = aws_dynamodb_table.idempotency.name
    event_timeline = aws_dynamodb_table.event_timeline.name
  }
}

output "table_arns" {
  description = "Map of logical name → DynamoDB table ARN"
  value = {
    orders         = aws_dynamodb_table.orders.arn
    payments       = aws_dynamodb_table.payments.arn
    inventory      = aws_dynamodb_table.inventory.arn
    idempotency    = aws_dynamodb_table.idempotency.arn
    event_timeline = aws_dynamodb_table.event_timeline.arn
  }
}
