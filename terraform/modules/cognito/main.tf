data "aws_caller_identity" "current" {}

# Domínio verificado manualmente no console SES — referenciado pelo ARN abaixo
# O recurso aws_ses_email_identity foi substituído pela verificação de domínio

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

  # ✅ Usar SES para envio de emails (sem limite diário, remetente personalizado)
  email_configuration {
    email_sending_account = "DEVELOPER"
    source_arn            = "arn:aws:ses:us-east-1:${data.aws_caller_identity.current.account_id}:identity/${var.ses_from_email}"
    from_email_address    = var.ses_from_email
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
    "ALLOW_ADMIN_USER_PASSWORD_AUTH",
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

# Referencia o email verificado no SES criado manualmente no console
data "aws_ses_email_identity" "from_email" {
  email = var.ses_from_email
}

# Policy que autoriza o Cognito a enviar emails via SES
resource "aws_ses_identity_policy" "cognito_send_email" {
  identity = var.ses_from_email
  name     = "cognito-send-email"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "AllowCognitoToSendEmail"
        Effect = "Allow"
        Principal = {
          Service = "email.cognito-idp.amazonaws.com"
        }
        Action   = "ses:SendEmail"
        Resource = "arn:aws:ses:us-east-1:${data.aws_caller_identity.current.account_id}:identity/${var.ses_from_email}"
      }
    ]
  })
}