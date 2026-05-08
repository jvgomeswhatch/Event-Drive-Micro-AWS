output "sqs_queue_urls" {
  description = "SQS queue URLs"
  value       = module.sqs.queue_urls
}

output "sqs_queue_arns" {
  description = "SQS queue ARNs"
  value       = module.sqs.queue_arns
}

output "sqs_dlq_urls" {
  description = "Dead Letter Queue URLs"
  value       = module.sqs.dlq_urls
}

output "sns_topic_arns" {
  description = "SNS topic ARNs"
  value       = module.sns.topic_arns
}

output "dynamodb_table_names" {
  description = "DynamoDB table names"
  value       = module.dynamodb.table_names
}

output "api_gateway_url" {
  description = "API Gateway invoke URL"
  value       = module.api_gateway.invoke_url
}

output "lambda_function_arns" {
  description = "Lambda function ARNs"
  value       = module.lambda.function_arns
}

output "iam_lambda_role_arn" {
  description = "IAM role ARN for Lambda execution"
  value       = module.iam.lambda_role_arn
}
