resource "aws_cognito_user_pool" "main" {
  name = "${var.app_name}-${var.environment}"

  username_attributes = ["email"]
  
  # ❌ REMOVER auto_verified_attributes
  # Isso faz o Cognito enviar email automático
  # auto_verified_attributes = ["email"]

  password_policy {
    minimum_length    = 8
    require_lowercase = true
    require_numbers   = true
    require_symbols   = true
    require_uppercase = true
  }

  schema {
    name                = "email"
    attribute_data_type = "String"
    required            = true
    mutable             = true
  }

  schema {
    name                = "name"
    attribute_data_type = "String"
    required            = true
    mutable             = true
  }

  account_recovery_setting {
    recovery_mechanism {
      name     = "verified_email"
      priority = 1
    }
  }

  # ✅ ADICIONAR: Configuração de email (sem SES por enquanto)
  email_configuration {
    email_sending_account = "COGNITO_DEFAULT"
  }

  tags = {
    Environment = var.environment
    Project     = "applywise"
  }
}

resource "aws_cognito_user_pool_client" "main" {
  name         = "${var.app_name}-client-${var.environment}"
  user_pool_id = aws_cognito_user_pool.main.id

  generate_secret = false

  explicit_auth_flows = [
    "ALLOW_USER_PASSWORD_AUTH",
    "ALLOW_REFRESH_TOKEN_AUTH",
    "ALLOW_USER_SRP_AUTH"
  ]

  refresh_token_validity = 30
  access_token_validity  = 1
  id_token_validity      = 1

  token_validity_units {
    refresh_token = "days"
    access_token  = "hours"
    id_token      = "hours"
  }
}