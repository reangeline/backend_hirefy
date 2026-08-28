terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Environment = var.environment
      Project     = "apply-wise"
      ManagedBy   = "terraform"
    }
  }
}



# DynamoDB Table
module "dynamodb" {
  source = "../../modules/dynamodb"

  environment = var.environment
  table_name  = var.dynamodb_table_name
}

# Cognito User Pool
module "cognito" {
  source = "../../modules/cognito"

  environment    = var.environment
  app_name       = var.app_name
  ses_from_email = var.ses_from_email
}

# Lambda Function
module "lambda" {
  source = "../../modules/lambda"

  environment   = var.environment
  function_name = "${var.app_name}-api-${var.environment}" # ← Adicionar sufixo
  handler       = "bootstrap"
  runtime       = "provided.al2"
  source_file   = "../../../deployment.zip"

  environment_variables = {
    DYNAMODB_TABLE            = module.dynamodb.table_name
    COGNITO_USER_POOL_ID      = module.cognito.user_pool_id
    COGNITO_CLIENT_ID         = module.cognito.client_id
    RESUMES_BUCKET_NAME       = module.s3.bucket_name
    STRIPE_SECRET_KEY            = var.stripe_secret_key
    STRIPE_WEBHOOK_SECRET        = var.stripe_webhook_secret
    STRIPE_PRICE_PREMIUM_MONTHLY = var.stripe_price_premium_monthly
    WEB_APP_BASE_URL             = var.web_app_base_url
    OPENAI_API_KEY               = var.openai_api_key
    SES_FROM_EMAIL               = var.ses_from_email
    ENVIRONMENT                  = var.environment
    REVENUECAT_WEBHOOK_SECRET    = var.revenuecat_webhook_secret
    OPTIMIZATION_QUEUE_URL       = module.optimization_queue.queue_url
    FIREBASE_CREDENTIALS_FILE    = var.firebase_credentials_file
    FIREBASE_PROJECT_ID          = var.firebase_project_id
  }

  dynamodb_table_arn        = module.dynamodb.table_arn
  cognito_pool_arn          = module.cognito.user_pool_arn
  ses_from_email            = var.ses_from_email
  revenuecat_webhook_secret = var.revenuecat_webhook_secret

  sqs_queue_arns = [module.optimization_queue.queue_arn]
  s3_bucket_arns = [module.s3.bucket_arn]


}

# Worker Lambda to consume optimization jobs
module "lambda_worker" {
  source = "../../modules/lambda"

  environment   = var.environment
  function_name = "${var.app_name}-worker-${var.environment}"
  handler       = "bootstrap"
  runtime       = "provided.al2"
  source_file   = "../../../deployment-worker.zip"
  lambda_timeout = 120

  environment_variables = {
    DYNAMODB_TABLE         = module.dynamodb.table_name
    OPENAI_API_KEY         = var.openai_api_key
    ENVIRONMENT            = var.environment
    OPTIMIZATION_QUEUE_URL = module.optimization_queue.queue_url
    FIREBASE_CREDENTIALS_FILE = var.firebase_credentials_file
    FIREBASE_PROJECT_ID       = var.firebase_project_id
  }

  dynamodb_table_arn        = module.dynamodb.table_arn
  cognito_pool_arn          = module.cognito.user_pool_arn
  ses_from_email            = var.ses_from_email
  revenuecat_webhook_secret = var.revenuecat_webhook_secret

  sqs_consume_arns = [module.optimization_queue.queue_arn]
}

resource "aws_lambda_event_source_mapping" "optimization_queue_to_worker" {
  event_source_arn = module.optimization_queue.queue_arn
  function_name    = module.lambda_worker.function_name
  batch_size       = 1
  enabled          = true
}

# SQS Queue for async optimization jobs
module "optimization_queue" {
  source = "../../modules/sqs"

  queue_name                 = var.optimization_queue_name
  visibility_timeout_seconds = 180
}

# API Gateway
module "api_gateway" {
  source = "../../modules/api_gateway"

  environment       = var.environment
  api_name          = "${var.app_name}-api"
  lambda_arn        = module.lambda.function_arn
  lambda_name       = module.lambda.function_name
  lambda_invoke_arn = module.lambda.invoke_arn
}

# S3 for Resume Storage (optional)
module "s3" {
  source = "../../modules/s3"

  environment = var.environment
  bucket_name = "applywise-resumes-dev-reangeline" # ← Nome único fixo
}


module "budget" {
  source = "../../modules/budget"

  environment = var.environment
  alert_email = "reangelinel@hotmail.com"
}