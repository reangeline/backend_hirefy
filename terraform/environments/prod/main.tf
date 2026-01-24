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
      Project     = "applywise"
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

  environment = var.environment
  app_name    = var.app_name
}

# Lambda Function
module "lambda" {
  source = "../../modules/lambda"

  environment           = var.environment
  function_name         = "${var.app_name}-api"
  handler              = "bootstrap"
  runtime              = "provided.al2"
  source_file          = "../../../deployment.zip"
  
  environment_variables = {
    DYNAMODB_TABLE          = module.dynamodb.table_name
    COGNITO_USER_POOL_ID    = module.cognito.user_pool_id
    COGNITO_CLIENT_ID       = module.cognito.client_id
    STRIPE_SECRET_KEY       = var.stripe_secret_key
    STRIPE_WEBHOOK_SECRET   = var.stripe_webhook_secret
    OPENAI_API_KEY          = var.openai_api_key
  }

  dynamodb_table_arn = module.dynamodb.table_arn
  cognito_pool_arn   = module.cognito.user_pool_arn
}

# API Gateway
module "api_gateway" {
  source = "../../modules/api_gateway"

  environment        = var.environment
  api_name           = "${var.app_name}-api"
  lambda_arn         = module.lambda.function_arn
  lambda_name        = module.lambda.function_name
  lambda_invoke_arn  = module.lambda.invoke_arn
}

# S3 for Resume Storage
module "s3" {
  source = "../../modules/s3"

  environment = var.environment
  bucket_name = "${var.app_name}-resumes-${var.environment}-prod"
}

module "budget" {
  source = "../../modules/budget"
  
  environment = var.environment
  alert_email = "reangelinel@hotmail.com"  # ← Seu email
}