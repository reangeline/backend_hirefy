aws_region          = "us-east-1"
environment         = "dev"
app_name            = "applywise"
dynamodb_table_name = "applywise-dev"

# Price ID do Stripe não é sensível (é só um identificador público, não uma chave) —
# pode ficar versionado. Sem isso o `terraform plan` trava esperando input interativo,
# que nunca chega no GitHub Actions (achado ao reativar o pipeline de CI/CD).
stripe_price_premium_monthly = "price_1U9DsjCgruCubp4jXMzIX9A7"