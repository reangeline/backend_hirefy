#!/bin/bash
# Cria o endpoint de webhook do Stripe (modo teste) apontando pro ambiente de dev na AWS, e
# já atualiza terraform/environments/dev/secrets.tfvars com o signing secret novo — o valor
# nunca é impresso no terminal, só escrito direto no arquivo (spec 007).
#
# Depois de rodar, é preciso fazer `terraform apply` de novo pra Lambda pegar o valor novo.

set -euo pipefail

WEBHOOK_URL="https://bo0aj4wdk2.execute-api.us-east-1.amazonaws.com/api/v1/webhooks/stripe"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SECRETS_FILE="${SCRIPT_DIR}/../terraform/environments/dev/secrets.tfvars"

if [ -z "${STRIPE_SECRET_KEY:-}" ]; then
  echo "Cole sua Stripe Secret Key (sk_test_... em modo teste):"
  read -rs STRIPE_SECRET_KEY
  echo
fi

echo "Criando webhook endpoint apontando pra ${WEBHOOK_URL}..."
RESPONSE=$(curl -sS --connect-timeout 10 --max-time 30 https://api.stripe.com/v1/webhook_endpoints \
  -u "${STRIPE_SECRET_KEY}:" \
  -d url="$WEBHOOK_URL" \
  -d "enabled_events[]=checkout.session.completed" \
  -d "enabled_events[]=customer.subscription.updated" \
  -d "enabled_events[]=customer.subscription.deleted" \
  -d "enabled_events[]=invoice.payment_succeeded" \
  -d "enabled_events[]=invoice.payment_failed") || {
  echo "Falha ao conectar à API do Stripe (erro de rede/conexão). Verifique sua internet e tente de novo."
  exit 1
}

WEBHOOK_ID=$(echo "$RESPONSE" | grep -o '"id": *"we_[^"]*"' | head -1 | grep -o 'we_[^"]*' || true)
if [ -z "$WEBHOOK_ID" ]; then
  echo "Falha ao criar o webhook. Resposta do Stripe:"
  echo "$RESPONSE"
  exit 1
fi

SIGNING_SECRET=$(echo "$RESPONSE" | grep -o '"secret": *"whsec_[^"]*"' | head -1 | grep -o 'whsec_[^"]*' || true)
if [ -z "$SIGNING_SECRET" ]; then
  echo "Webhook criado (${WEBHOOK_ID}), mas a resposta não trouxe o signing secret. Confira manualmente no dashboard do Stripe (Developers > Webhooks) e atualize stripe_webhook_secret em ${SECRETS_FILE} à mão."
  exit 1
fi

if [ ! -f "$SECRETS_FILE" ]; then
  echo "Webhook criado (${WEBHOOK_ID}), mas não achei ${SECRETS_FILE} pra atualizar sozinho."
  echo "Atualize stripe_webhook_secret manualmente com o valor mostrado no dashboard do Stripe."
  exit 1
fi

sed -i.bak "s#^stripe_webhook_secret.*#stripe_webhook_secret = \"${SIGNING_SECRET}\"#" "$SECRETS_FILE"
rm -f "${SECRETS_FILE}.bak"

echo "Webhook criado (${WEBHOOK_ID}) e secrets.tfvars atualizado com o signing secret novo."
echo "Agora rode o deploy de novo (terraform apply) pra Lambda de dev pegar o valor."
