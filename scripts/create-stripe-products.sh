#!/bin/bash

# Script para criar produtos e preços no Stripe
# Execute uma vez para configurar os planos de assinatura

STRIPE_KEY=$STRIPE_SECRET_KEY

echo "Creating Stripe products and prices..."

# Criar produto Basic
BASIC_PRODUCT=$(stripe products create \
  --name="ApplyWise Basic" \
  --description="Basic plan with limited resume optimizations" \
  -k $STRIPE_KEY)

BASIC_PRODUCT_ID=$(echo $BASIC_PRODUCT | jq -r '.id')

# Criar preço para Basic (exemplo: $9.99/mês)
BASIC_PRICE=$(stripe prices create \
  --product=$BASIC_PRODUCT_ID \
  --unit-amount=999 \
  --currency=usd \
  --recurring[interval]=month \
  -k $STRIPE_KEY)

BASIC_PRICE_ID=$(echo $BASIC_PRICE | jq -r '.id')

echo "Basic Plan Price ID: $BASIC_PRICE_ID"

# Criar produto Premium
PREMIUM_PRODUCT=$(stripe products create \
  --name="ApplyWise Premium" \
  --description="Premium plan with unlimited resume optimizations" \
  -k $STRIPE_KEY)

PREMIUM_PRODUCT_ID=$(echo $PREMIUM_PRODUCT | jq -r '.id')

# Criar preço para Premium (exemplo: $19.99/mês)
PREMIUM_PRICE=$(stripe prices create \
  --product=$PREMIUM_PRODUCT_ID \
  --unit-amount=1999 \
  --currency=usd \
  --recurring[interval]=month \
  -k $STRIPE_KEY)

PREMIUM_PRICE_ID=$(echo $PREMIUM_PRICE | jq -r '.id')

echo "Premium Plan Price ID: $PREMIUM_PRICE_ID"

echo ""
echo "Add these to your .env file:"
echo "STRIPE_PRICE_BASIC_MONTHLY=$BASIC_PRICE_ID"
echo "STRIPE_PRICE_PREMIUM_MONTHLY=$PREMIUM_PRICE_ID"
