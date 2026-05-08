#!/bin/bash
set -e

echo "==> Linting services (Go)..."
for service in order-service payment-service inventory-service notification-service; do
  echo "  --> $service"
  (cd "services/$service" && go vet ./... && echo "    vet: ok")
done

echo "==> Linting frontend..."
(cd frontend && npm run lint 2>/dev/null || true)

echo "==> Lint done."
