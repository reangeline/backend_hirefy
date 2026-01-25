variable "environment" {
  description = "Environment name"
  type        = string
}

variable "function_name" {
  description = "Lambda function name"
  type        = string
}

variable "handler" {
  description = "Lambda handler"
  type        = string
}

variable "runtime" {
  description = "Lambda runtime"
  type        = string
}

variable "source_file" {
  description = "Path to deployment package"
  type        = string
}

variable "environment_variables" {
  description = "Environment variables"
  type        = map(string)
  default     = {}
}

variable "dynamodb_table_arn" {
  description = "DynamoDB table ARN"
  type        = string
}

variable "cognito_pool_arn" {
  description = "Cognito user pool ARN"
  type        = string
}

variable "ses_from_email" {
  description = "Email address to send from (must be verified in SES)"
  type        = string
}