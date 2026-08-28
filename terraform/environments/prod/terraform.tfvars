aws_region          = "us-east-1"
environment         = "prod"
app_name            = "applywise"
dynamodb_table_name = "applywise-prod"

# Price ID não é sensível (identificador público, não uma chave). PLACEHOLDER: é o price
# antigo do modelo Free/Pro (pré-spec-007) em modo teste — checkout Premium não funciona
# de verdade em produção até o Stripe live ser configurado (ver pending_prod_stripe_launch_setup).
stripe_price_premium_monthly = "price_1SsqGYLZ2WOLEJxZktW6EQqe"