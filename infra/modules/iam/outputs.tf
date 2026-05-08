output "lambda_role_arn" {
  description = "ARN of the shared Lambda execution role"
  value       = aws_iam_role.lambda_exec.arn
}

output "lambda_role_name" {
  description = "Name of the shared Lambda execution role"
  value       = aws_iam_role.lambda_exec.name
}

output "service_role_arns" {
  description = "Map of service name → IAM role ARN"
  value = {
    for k, r in aws_iam_role.service : k => r.arn
  }
}

output "service_role_names" {
  description = "Map of service name → IAM role name"
  value = {
    for k, r in aws_iam_role.service : k => r.name
  }
}
