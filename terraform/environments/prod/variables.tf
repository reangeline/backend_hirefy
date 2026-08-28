variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "prod"
}

variable "app_name" {
  description = "Application name"
  type        = string
  default     = "applywise"
}

variable "dynamodb_table_name" {
  description = "DynamoDB table name"
  type        = string
  default     = "applywise-prod"
}

variable "openai_api_key" {
  description = "OpenAI API Key"
  type        = string
  sensitive   = true
}

variable "stripe_secret_key" {
  description = "Stripe Secret Key"
  type        = string
  sensitive   = true
}

variable "stripe_webhook_secret" {
  description = "Stripe Webhook Secret"
  type        = string
  sensitive   = true
}

variable "stripe_price_premium_monthly" {
  description = "Stripe Price ID do plano Premium (US$19,99/mês) — spec 007. Placeholder de teste até o Stripe live de produção ser configurado (ver .spec de billing)."
  type        = string
}

variable "ses_from_email" {
  description = "Email to send from"
  type        = string
  default     = "contact@hirefy.careers"
}

variable "optimization_queue_name" {
  description = "SQS queue name for async resume optimization"
  type        = string
  default     = "applywise-optimization-prod"
}

variable "firebase_credentials_file" {
  description = "Path to Firebase service account JSON inside the Lambda package or layer"
  type        = string
  default     = "/var/task/firebase_credentials.json"
}

variable "firebase_project_id" {
  description = "Firebase project ID (used to init FCM)"
  type        = string
  default     = "applywise-35cc7"
}

variable "revenuecat_webhook_secret" {
  description = "RevenueCat webhook secret"
  type        = string
  sensitive   = true
  default     = ""
}