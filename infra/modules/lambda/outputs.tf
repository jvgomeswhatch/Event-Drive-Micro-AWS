output "function_arns" {
  description = "Map of function name → ARN"
  value = {
    for k, f in aws_lambda_function.this : k => f.arn
  }
}

output "function_names" {
  description = "Map of function name → full function name"
  value = {
    for k, f in aws_lambda_function.this : k => f.function_name
  }
}

output "invoke_arns" {
  description = "Map of function name → invoke ARN (used by API Gateway)"
  value = {
    for k, f in aws_lambda_function.this : k => f.invoke_arn
  }
}
