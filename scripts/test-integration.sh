#!/bin/bash
set -e

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "==> Testes de integração (requer LocalStack + serviços em execução)"
echo "    Use 'make up' antes de rodar este script."
echo ""

# Verifica se LocalStack está acessível
if ! curl -sf "$LOCALSTACK_ENDPOINT/_localstack/health" >/dev/null 2>&1 &&
   ! curl -sf "http://localhost:4566/_localstack/health" >/dev/null 2>&1; then
  echo "  [ERRO] LocalStack não está acessível. Execute: make up"
  exit 1
fi

LOCALSTACK_ENDPOINT="${LOCALSTACK_ENDPOINT:-http://localhost:4566}"
export LOCALSTACK_ENDPOINT

echo "==> Rodando testes de integração Go com -tags integration"
(cd "$ROOT/tests/integration" && \
  go test ./... -v -count=1 -timeout 120s -tags integration \
    -run "TestFluxo|TestDLQ|TestIdempotencia")

echo ""
echo "==> Testes de integração concluídos."
