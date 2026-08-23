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

variable "revenuecat_webhook_secret" {
  description = "RevenueCat webhook secret for signature validation"
  type        = string
  sensitive   = true
}

variable "lambda_timeout" {
  description = "Lambda timeout in seconds"
  type        = number
  default     = 29
}

variable "sqs_queue_arns" {
  description = "List of SQS queue ARNs the function can send messages to"
  type        = list(string)
  default     = []
}

variable "sqs_consume_arns" {
  description = "List of SQS queue ARNs the function can consume messages from"
  type        = list(string)
  default     = []
}

variable "s3_bucket_arns" {
  description = "List of S3 bucket ARNs the function can delete objects from"
  type        = list(string)
  default     = []
}