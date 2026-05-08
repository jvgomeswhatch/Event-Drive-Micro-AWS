output "rest_api_id" {
  value = aws_api_gateway_rest_api.this.id
}

output "stage_name" {
  value = aws_api_gateway_stage.this.stage_name
}

output "invoke_url" {
  description = "Base URL to call the API (e.g. http://localhost:4566/restapis/{id}/dev/_user_request_)"
  value       = aws_api_gateway_stage.this.invoke_url
}
