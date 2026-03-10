variable "queue_name" {
  description = "SQS queue name"
  type        = string
}

variable "message_retention_seconds" {
  description = "Retention period"
  type        = number
  default     = 345600 # 4 days
}

variable "visibility_timeout_seconds" {
  description = "Visibility timeout"
  type        = number
  default     = 120
}

variable "receive_wait_time_seconds" {
  description = "Long polling wait time"
  type        = number
  default     = 10
}
