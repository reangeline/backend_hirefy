#!/bin/bash
# Atualiza stripe_secret_key em terraform/environments/dev/secrets.tfvars sem nunca imprimir
# o valor no terminal. Achado spec 007: a key salva antes desta sessão estava inválida/
# revogada (401 "Invalid API Key provided" nos logs do CloudWatch da Lambda de dev).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SECRETS_FILE="${SCRIPT_DIR}/../terraform/environments/dev/secrets.tfvars"

if [ ! -f "$SECRETS_FILE" ]; then
  echo "Não achei ${SECRETS_FILE}"
  exit 1
fi

echo "Cole a Stripe Secret Key válida (sk_test_... em modo teste — a mesma que você usou nos scripts anteriores):"
read -rs STRIPE_SECRET_KEY
echo

if [ -z "$STRIPE_SECRET_KEY" ]; then
  echo "Nenhum valor colado, nada foi alterado."
  exit 1
fi

sed -i.bak "s#^stripe_secret_key.*#stripe_secret_key     = \"${STRIPE_SECRET_KEY}\"#" "$SECRETS_FILE"
rm -f "${SECRETS_FILE}.bak"

echo "secrets.tfvars atualizado. Rode o deploy (terraform apply) de novo pra Lambda pegar o valor."
