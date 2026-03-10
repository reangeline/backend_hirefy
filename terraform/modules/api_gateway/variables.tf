variable "api_name" {
  description = "Name of the API Gateway"
  type        = string
}

variable "lambda_invoke_arn" {
  description = "Invoke ARN of the Lambda function"
  type        = string
}

variable "lambda_name" {
  description = "Name of the Lambda function"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "lambda_arn" {
  description = "The ARN of the Lambda function to integrate with API Gateway"
  type        = string
}
