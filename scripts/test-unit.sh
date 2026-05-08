#!/bin/bash
set -e

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "==> Testes unitários — Go (serviços)"
for service in order-service payment-service inventory-service notification-service; do
  echo "  --> $service"
  (cd "$ROOT/services/$service" && go test ./... -v -count=1 -timeout 30s)
done

echo ""
echo "==> Testes de contrato — Go"
(cd "$ROOT/tests/contract" && go test ./... -v -count=1)

echo ""
echo "==> Testes unitários — Frontend (Vitest)"
if command -v node >/dev/null 2>&1; then
  (cd "$ROOT/frontend" && npm test)
else
  echo "  [skip] node não encontrado"
fi

echo ""
echo "==> Testes unitários concluídos."
