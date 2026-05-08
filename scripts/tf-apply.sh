#!/bin/sh
set -e

INFRA_DIR="/infra/envs/dev"
TF_ARGS="${TF_ARGS:-apply}"

echo "==> Waiting for LocalStack to be ready..."
until wget -qO- http://localstack:4566/_localstack/health 2>/dev/null | grep -qE '"sqs"[[:space:]]*:[[:space:]]*"(running|available)"'; do
  echo "    LocalStack not ready yet, retrying in 3s..."
  sleep 3
done

echo "==> LocalStack is ready."
echo "==> Running Terraform $TF_ARGS in $INFRA_DIR"

cd "$INFRA_DIR"

terraform init -backend=false -input=false

if [ "$TF_ARGS" = "destroy" ]; then
  terraform destroy -auto-approve -input=false
else
  terraform apply -auto-approve -input=false
fi

echo "==> Terraform done."
