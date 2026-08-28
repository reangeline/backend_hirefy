#!/bin/bash
# Cria o produto e o preço reais do "Hirefy Premium" no Stripe (US$19,99/mês, recorrente).
# Substitui os dois scripts antigos e conflitantes (setup-stripe-products.sh,
# create-stripe-products.sh — nenhum rodado, nenhum batia com a marca Hirefy nem com o
# plano decidido na spec 007). Rode você mesmo — a secret key nunca sai da sua máquina.

set -euo pipefail

GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

if [ -z "${STRIPE_SECRET_KEY:-}" ]; then
  echo "Cole sua Stripe Secret Key (sk_test_... em modo teste):"
  read -rs STRIPE_SECRET_KEY
  echo
fi

echo -e "${BLUE}Criando produto 'Hirefy Premium'...${NC}"
PRODUCT=$(curl -sS --connect-timeout 10 --max-time 30 https://api.stripe.com/v1/products \
  -u "${STRIPE_SECRET_KEY}:" \
  -d name="Hirefy Premium" \
  -d description="Otimizações de currículo e coach de IA ilimitados") || {
  echo "Falha ao conectar à API do Stripe (erro de rede/conexão). Verifique sua internet e tente de novo."
  exit 1
}

PRODUCT_ID=$(echo "$PRODUCT" | grep -o '"id": *"prod_[^"]*"' | head -1 | grep -o 'prod_[^"]*' || true)
if [ -z "$PRODUCT_ID" ]; then
  echo "Falha ao criar o produto. Resposta do Stripe:"
  echo "$PRODUCT"
  exit 1
fi
echo -e "${GREEN}Produto criado: ${PRODUCT_ID}${NC}"

echo -e "${BLUE}Criando preço mensal (US\$19,99)...${NC}"
PRICE=$(curl -sS --connect-timeout 10 --max-time 30 https://api.stripe.com/v1/prices \
  -u "${STRIPE_SECRET_KEY}:" \
  -d product="$PRODUCT_ID" \
  -d unit_amount=1999 \
  -d currency=usd \
  -d "recurring[interval]=month") || {
  echo "Falha ao conectar à API do Stripe (erro de rede/conexão). Verifique sua internet e tente de novo."
  exit 1
}

PRICE_ID=$(echo "$PRICE" | grep -o '"id": *"price_[^"]*"' | head -1 | grep -o 'price_[^"]*' || true)
if [ -z "$PRICE_ID" ]; then
  echo "Falha ao criar o preço. Resposta do Stripe:"
  echo "$PRICE"
  exit 1
fi

echo -e "${GREEN}Preço criado: ${PRICE_ID}${NC}"
echo
echo "Adicione ao seu .env (o valor do price_id não é sensível, pode compartilhar):"
echo "STRIPE_PRICE_PREMIUM_MONTHLY=${PRICE_ID}"
