variable "function_name" {
  description = "Name of the Lambda function"
  type        = string
}

variable "handler" {
  description = "Lambda function handler"
  type        = string
}

variable "runtime" {
  description = "Lambda runtime"
  type        = string
}

variable "source_file" {
  description = "Path to the deployment package"
  type        = string
}

variable "environment_variables" {
  description = "Environment variables for Lambda"
  type        = map(string)
  default     = {}
}

variable "dynamodb_table_arn" {
  description = "ARN of the DynamoDB table"
  type        = string
}

variable "cognito_pool_arn" {
  description = "ARN of the Cognito User Pool"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}