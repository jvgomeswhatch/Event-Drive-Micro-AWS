#!/bin/bash
# Creates the S3 bucket used as Terraform remote state (LocalStack simulation).
# Run once before enabling the S3 backend block in main.tf.
set -e

ENDPOINT="${LOCALSTACK_ENDPOINT:-http://localhost:4566}"
BUCKET="terraform-state-dev"
REGION="${AWS_DEFAULT_REGION:-us-east-1}"

echo "==> Creating S3 state bucket: $BUCKET"

aws --endpoint-url="$ENDPOINT" --region="$REGION" s3api create-bucket \
  --bucket "$BUCKET" \
  --create-bucket-configuration LocationConstraint="$REGION" 2>/dev/null || \
  echo "    Bucket already exists, skipping."

aws --endpoint-url="$ENDPOINT" --region="$REGION" s3api put-bucket-versioning \
  --bucket "$BUCKET" \
  --versioning-configuration Status=Enabled

echo "==> State bucket ready: s3://$BUCKET"
echo "    Uncomment the backend block in infra/envs/dev/main.tf to enable remote state."
