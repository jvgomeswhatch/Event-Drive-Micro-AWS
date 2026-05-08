output "topic_arns" {
  description = "Map of topic base name → ARN"
  value = {
    for k, t in aws_sns_topic.this : k => t.arn
  }
}

output "topic_names" {
  description = "Map of topic base name → full name"
  value = {
    for k, t in aws_sns_topic.this : k => t.name
  }
}
