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

  environment = var.environment
  app_name    = var.app_name
}

# Lambda Function
module "lambda" {
  source = "../../modules/lambda"

  environment           = var.environment
  function_name         = "${var.app_name}-api-${var.environment}"  # ← Adicionar sufixo
  handler              = "bootstrap"
  runtime              = "provided.al2"
  source_file          = "../../../deployment.zip"
  
  environment_variables = {
    DYNAMODB_TABLE          = module.dynamodb.table_name
    COGNITO_USER_POOL_ID    = module.cognito.user_pool_id
    COGNITO_CLIENT_ID       = module.cognito.client_id
    STRIPE_SECRET_KEY       = var.stripe_secret_key
    STRIPE_WEBHOOK_SECRET   = var.stripe_webhook_secret
    STRIPE_FREE_PRICE_ID    = var.stripe_free_price_id
    STRIPE_PRO_PRICE_ID     = var.stripe_pro_price_id   
    OPENAI_API_KEY          = var.openai_api_key
    SES_FROM_EMAIL            = var.ses_from_email 
    ENVIRONMENT               = var.environment
    REVENUECAT_WEBHOOK_SECRET = var.revenuecat_webhook_secret
  }

  dynamodb_table_arn = module.dynamodb.table_arn
  cognito_pool_arn   = module.cognito.user_pool_arn
  ses_from_email           = var.ses_from_email
  revenuecat_webhook_secret = var.revenuecat_webhook_secret


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

# S3 for Resume Storage (optional)
module "s3" {
  source = "../../modules/s3"

  environment = var.environment
  bucket_name = "applywise-resumes-dev-reangeline"  # ← Nome único fixo
}


module "budget" {
  source = "../../modules/budget"
  
  environment = var.environment
  alert_email = "reangelinel@hotmail.com"
}