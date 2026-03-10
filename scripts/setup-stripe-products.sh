#!/bin/bash

# Cores para output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🔧 Setting up Stripe Products...${NC}\n"

# Ler a chave do Stripe
if [ -z "$STRIPE_SECRET_KEY" ]; then
    echo "Enter your Stripe Secret Key (sk_test_...):"
    read -s STRIPE_SECRET_KEY
fi

# Criar produto Free
echo -e "\n${BLUE}Creating Free Product...${NC}"
FREE_PRODUCT=$(curl -s https://api.stripe.com/v1/products \
  -u "$STRIPE_SECRET_KEY:" \
  -d name="ApplyWise Free" \
  -d description="Free tier with basic features")

FREE_PRODUCT_ID=$(echo $FREE_PRODUCT | grep -o '"id":"prod_[^"]*' | cut -d'"' -f4)
echo -e "${GREEN}✅ Free Product ID: $FREE_PRODUCT_ID${NC}"

# Criar preço Free (R$ 0)
FREE_PRICE=$(curl -s https://api.stripe.com/v1/prices \
  -u "$STRIPE_SECRET_KEY:" \
  -d product="$FREE_PRODUCT_ID" \
  -d unit_amount=0 \
  -d currency=usd \
  -d "recurring[interval]=month")

FREE_PRICE_ID=$(echo $FREE_PRICE | grep -o '"id":"price_[^"]*' | cut -d'"' -f4)
echo -e "${GREEN}✅ Free Price ID: $FREE_PRICE_ID${NC}"

# Criar produto Pro
echo -e "\n${BLUE}Creating Pro Product...${NC}"
PRO_PRODUCT=$(curl -s https://api.stripe.com/v1/products \
  -u "$STRIPE_SECRET_KEY:" \
  -d name="ApplyWise Pro" \
  -d description="Professional plan with unlimited optimizations")

PRO_PRODUCT_ID=$(echo $PRO_PRODUCT | grep -o '"id":"prod_[^"]*' | cut -d'"' -f4)
echo -e "${GREEN}✅ Pro Product ID: $PRO_PRODUCT_ID${NC}"

# Criar preço Pro (R$ 9.99)
PRO_PRICE=$(curl -s https://api.stripe.com/v1/prices \
  -u "$STRIPE_SECRET_KEY:" \
  -d product="$PRO_PRODUCT_ID" \
  -d unit_amount=999 \
  -d currency=usd \
  -d "recurring[interval]=month")

PRO_PRICE_ID=$(echo $PRO_PRICE | grep -o '"id":"price_[^"]*' | cut -d'"' -f4)
echo -e "${GREEN}✅ Pro Price ID: $PRO_PRICE_ID${NC}"

# Salvar em arquivo
cat > stripe-config.env << EOL
# Stripe Product Configuration
# Generated at $(date)

# Free Tier
STRIPE_FREE_PRODUCT_ID=$FREE_PRODUCT_ID
STRIPE_FREE_PRICE_ID=$FREE_PRICE_ID

# Pro Plan
STRIPE_PRO_PRODUCT_ID=$PRO_PRODUCT_ID
STRIPE_PRO_PRICE_ID=$PRO_PRICE_ID
EOL

echo -e "\n${GREEN}✅ Configuration saved to stripe-config.env${NC}"
echo -e "\n${BLUE}Add these to your Lambda environment variables!${NC}"
cat stripe-config.env
