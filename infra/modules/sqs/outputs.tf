output "queue_urls" {
  description = "Map of service → main queue URL"
  value = {
    for svc, q in aws_sqs_queue.main : "${svc}-queue" => q.url
  }
}

output "queue_arns" {
  description = "Map of queue name → ARN (used by SNS subscriptions and IAM)"
  value = merge(
    { for svc, q in aws_sqs_queue.main : "${svc}-queue" => q.arn },
    { for svc, q in aws_sqs_queue.dlq  : "${svc}-dlq"   => q.arn }
  )
}

output "dlq_urls" {
  description = "Map of service → DLQ URL"
  value = {
    for svc, q in aws_sqs_queue.dlq : "${svc}-dlq" => q.url
  }
}

output "dlq_arns" {
  description = "Map of service → DLQ ARN"
  value = {
    for svc, q in aws_sqs_queue.dlq : "${svc}-dlq" => q.arn
  }
}
