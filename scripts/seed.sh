#!/bin/bash
set -e

ENDPOINT="${LOCALSTACK_ENDPOINT:-http://localhost:4566}"
REGION="${AWS_DEFAULT_REGION:-us-east-1}"
ENV="${ENVIRONMENT:-dev}"

export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-test}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-test}"

echo "==> Aguardando tabela inventory-$ENV existir..."
until aws --endpoint-url="$ENDPOINT" --region="$REGION" dynamodb describe-table \
  --table-name "inventory-$ENV" >/dev/null 2>&1; do
  echo "    Tabela não encontrada, aguardando 3s..."
  sleep 3
done

echo "==> Seeding inventory table (inventory-$ENV)..."

aws --endpoint-url="$ENDPOINT" --region="$REGION" dynamodb put-item \
  --table-name "inventory-$ENV" \
  --item '{
    "productId": {"S": "prod-001"},
    "name": {"S": "Mechanical Keyboard"},
    "quantity": {"N": "50"},
    "reserved": {"N": "0"},
    "price": {"N": "299.99"}
  }'

aws --endpoint-url="$ENDPOINT" --region="$REGION" dynamodb put-item \
  --table-name "inventory-$ENV" \
  --item '{
    "productId": {"S": "prod-002"},
    "name": {"S": "Wireless Mouse"},
    "quantity": {"N": "120"},
    "reserved": {"N": "0"},
    "price": {"N": "79.99"}
  }'

aws --endpoint-url="$ENDPOINT" --region="$REGION" dynamodb put-item \
  --table-name "inventory-$ENV" \
  --item '{
    "productId": {"S": "prod-003"},
    "name": {"S": "USB-C Hub"},
    "quantity": {"N": "30"},
    "reserved": {"N": "0"},
    "price": {"N": "49.99"}
  }'

echo "==> Seed complete."
