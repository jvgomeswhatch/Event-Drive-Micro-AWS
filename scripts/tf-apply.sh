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

if [ "$TF_ARGS" = "destroy" ]; then
  terraform init -backend=false -input=false
  terraform destroy -auto-approve -input=false
else
  attempt=1
  max_attempts=5
  while [ $attempt -le $max_attempts ]; do
    echo "==> Terraform init+apply attempt $attempt/$max_attempts..."
    # providers já estão na imagem Docker; init apenas sincroniza módulos e lock file
    if terraform init -backend=false -input=false && terraform apply -auto-approve -input=false; then
      echo "==> Terraform done."
      exit 0
    fi
    echo "==> Attempt $attempt failed, waiting 10s before retry..."
    attempt=$((attempt + 1))
    sleep 10
  done
  echo "==> Terraform apply failed after $max_attempts attempts."
  exit 1
fi
