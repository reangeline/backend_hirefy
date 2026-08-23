variable "app_name" {
  description = "Application name"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "ses_from_email" {
  description = "Email address verified in SES used to send emails from Cognito"
  type        = string
  default     = "contact@hirefy.careers"
}

variable "ses_domain" {
  description = "Domain verified in SES (e.g. hirefy.careers)"
  type        = string
  default     = "hirefy.careers"
}