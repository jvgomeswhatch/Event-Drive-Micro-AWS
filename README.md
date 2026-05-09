# Event-Driven Microservices Platform

Plataforma de processamento de pedidos event-driven, rodando 100% local com Terraform + Docker + LocalStack simulando AWS.

## Arquitetura

```
                       ┌──────────────────────────────────┐
                       │    Frontend (React + Vite)       │
                       │Dashboard http://localhost:5173 │
                       └─────────────┬────────────────────┘
                                     │ HTTP REST
                       ┌─────────────▼────────────────────┐
                       │           order-service          │
                       │  Go HTTP server — porta 3001     │
                       │  - Valida input + JWT            │
                       │  - Persiste pedido (DynamoDB)    │
                       │  - Publica order.created → SNS   │
                       └──────────────┬───────────────────┘
                                      │ SNS fan-out
              ┌───────────────────────▼──────────────────────────┐
              │              SNS: order-events-dev               │
              └──────┬───────────────────────────────────────────┘
                     │ SQS: payment-queue-dev
       ┌─────────────▼──────────────┐
       │       payment-service      │
       │  Go consumer — porta 3002  │
       │  - Processa pagamento      │
       │  - payment.succeeded/failed│
       │  → SNS: payment-events-dev │
       └──────┬──────────┬──────────┘
              │          │ SQS: inventory-queue-dev + notification-queue-dev
              │    ┌─────▼──────────────────────┐
              │    │  inventory-service         │
              │    │  Go consumer — 3003        │
              │    │  Reserva estoque           │
              │    │  inventory.reserved        │
              │    │SNS:inventory-events-dev    │
              │    └─────┬──────────────────────┘
              │          │ SQS: notification-queue-dev
       ┌──────▼──────────▼──────────┐
       │     notification-service   │
       │  Go consumer — porta 3004  │
       │  - Agrega eventos          │
       │  - Atualiza status DynamoDB│
       │  - Simula email/websocket  │
       └────────────────────────────┘
```

### Infraestrutura local

| Serviço | Porta | Função |
|---|---|---|
| LocalStack | 4566 | AWS simulado (SQS, SNS, DynamoDB, Lambda, IAM) |
| Grafana | 3000 | Dashboard de logs |
| Loki | 3100 | Agregador de logs |
| Promtail | — | Coleta logs dos containers |

---

## Stack

- **Backend**: Go 1.24 — todos os 4 microserviços
- **Frontend**: React 18 + TypeScript + Vite + TanStack Query
- **Infra**: Terraform 1.8 + LocalStack 3.4
- **Mensageria**: SQS (filas por serviço) + SNS (fan-out de eventos)
- **Banco**: DynamoDB (tabela por serviço)
- **Auth**: JWT HS256 simulado
- **Observabilidade**: JSON logs estruturados + Correlation ID + Loki/Grafana

---

## Quick Start

**Pré-requisitos**: Docker Desktop rodando. Só isso.

```bash
# 1. Sobe tudo: LocalStack, Terraform, 4 serviços Go, frontend, Grafana
make up

# 2. Popula inventário com produtos de exemplo
make seed

# 3. Abre o dashboard
# http://localhost:5173
```

O `make up` faz tudo: builda as imagens Go, sobe o LocalStack, roda o Terraform (cria filas SQS, tópicos SNS, tabelas DynamoDB) e sobe os serviços. Não precisa instalar Go, Node, AWS CLI nem Terraform na sua máquina.

---

## Como testar

### Health de todos os serviços

```bash
make status

# Ou direto:
curl http://localhost:3001/health
curl http://localhost:3002/health
curl http://localhost:3003/health
curl http://localhost:3004/health
```

### Fluxo completo via curl

```bash
# 1. Pegar token JWT
TOKEN=$(curl -s -X POST http://localhost:3001/auth/token -H "Content-Type: application/json" -d "{\"customerId\":\"user-123\",\"role\":\"customer\"}" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

# 2. Criar pedido (fluxo feliz — prod-001 tem estoque após make seed)
curl -s -X POST http://localhost:3001/orders -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -H "X-Idempotency-Key: pedido-001" -d "{\"customerId\":\"user-123\",\"items\":[{\"productId\":\"prod-001\",\"quantity\":1,\"unitPrice\":299.99}],\"simulateFailure\":false}"

# 3. Consultar pedido (substitua pelo orderId retornado)
curl -s http://localhost:3001/orders/<ORDER_ID> -H "Authorization: Bearer $TOKEN"

# 4. Listar pedidos do cliente
curl -s "http://localhost:3001/orders?customerId=user-123" -H "Authorization: Bearer $TOKEN"
```

### Simular falha de pagamento

```bash
curl -s -X POST http://localhost:3001/orders -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -H "X-Idempotency-Key: pedido-falha-001" -d "{\"customerId\":\"user-123\",\"items\":[{\"productId\":\"prod-002\",\"quantity\":1,\"unitPrice\":79.99}],\"simulateFailure\":true}"
```

### Simular falta de estoque

```bash
# prod-999 não existe no inventário
curl -s -X POST http://localhost:3001/orders -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -H "X-Idempotency-Key: pedido-sem-estoque-001" -d "{\"customerId\":\"user-123\",\"items\":[{\"productId\":\"prod-999\",\"quantity\":1,\"unitPrice\":10.00}],\"simulateFailure\":false}"
```

### Testar idempotência

```bash
# Enviar duas vezes com a mesma chave — a segunda retorna {"message":"duplicate request","cached":true}
curl -s -X POST http://localhost:3001/orders -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -H "X-Idempotency-Key: chave-fixa-xyz" -d "{\"customerId\":\"user-123\",\"items\":[{\"productId\":\"prod-001\",\"quantity\":1,\"unitPrice\":299.99}],\"simulateFailure\":false}"
```

### Acompanhar o fluxo em tempo real

```bash
# Logs de todos os serviços Go juntos
make logs

# Um serviço específico
docker logs -f payment-service

# Filtrar pelo correlationId de um pedido específico
docker logs payment-service 2>&1 | grep "<correlationId>"
```

### Grafana

Acesse `http://localhost:3000` → Explore → selecione Loki → query:
```
{service="order-service"}
{service=~"order-service|payment-service"} |= "correlationId"
```

### Inspecionar infraestrutura AWS (LocalStack)

```bash
# Filas SQS
aws --endpoint-url=http://localhost:4566 --region=us-east-1 sqs list-queues

# Dead Letter Queue (mensagens com falha)
aws --endpoint-url=http://localhost:4566 --region=us-east-1 sqs receive-message --queue-url http://localhost:4566/000000000000/order-dlq-dev

# Tópicos SNS
aws --endpoint-url=http://localhost:4566 --region=us-east-1 sns list-topics

# Tabela de pedidos
aws --endpoint-url=http://localhost:4566 --region=us-east-1 dynamodb scan --table-name orders-dev

# Inventário
aws --endpoint-url=http://localhost:4566 --region=us-east-1 dynamodb scan --table-name inventory-dev
```

---

## Banco de dados (DynamoDB via LocalStack)

Cinco tabelas, cada uma com responsabilidade isolada por serviço.

### Tabelas

| Tabela | Dono | Chave primária | GSIs |
|---|---|---|---|
| `orders-dev` | order-service | `orderId` | `customerId-createdAt`, `status-createdAt` |
| `payments-dev` | payment-service | `paymentId` | `orderId-index` |
| `inventory-dev` | inventory-service | `productId` | — |
| `idempotency-dev` | todos os serviços | `idempotencyKey` | — |
| `event-timeline-dev` | inventory + notification | `orderId` + `eventId` | — |

**`idempotency-dev`** é compartilhada: cada serviço escreve uma chave `{servico}#{messageId}` antes de processar. Se a chave já existe, descarta a mensagem sem reprocessar. TTL de 24h — limpa sozinha.

**`event-timeline-dev`** armazena o histórico de eventos por pedido para o frontend exibir a timeline (qual serviço processou o quê e quando).

### Consultar o banco direto

```bash
# Todos os pedidos
aws --endpoint-url=http://localhost:4566 --region=us-east-1 dynamodb scan --table-name orders-dev

# Um pedido específico
aws --endpoint-url=http://localhost:4566 --region=us-east-1 dynamodb get-item --table-name orders-dev --key "{\"orderId\":{\"S\":\"<ORDER_ID>\"}}"

# Pedidos de um cliente (via GSI)
aws --endpoint-url=http://localhost:4566 --region=us-east-1 dynamodb query --table-name orders-dev --index-name customerId-createdAt-index --key-condition-expression "customerId = :c" --expression-attribute-values "{\":c\":{\"S\":\"user-123\"}}"

# Estoque atual
aws --endpoint-url=http://localhost:4566 --region=us-east-1 dynamodb scan --table-name inventory-dev

# Pagamentos
aws --endpoint-url=http://localhost:4566 --region=us-east-1 dynamodb scan --table-name payments-dev

# Timeline de eventos de um pedido
aws --endpoint-url=http://localhost:4566 --region=us-east-1 dynamodb query --table-name event-timeline-dev --key-condition-expression "orderId = :o" --expression-attribute-values "{\":o\":{\"S\":\"<ORDER_ID>\"}}"

# Chaves de idempotência ativas
aws --endpoint-url=http://localhost:4566 --region=us-east-1 dynamodb scan --table-name idempotency-dev
```

### Resetar os dados

```bash
# Limpa tudo (LocalStack + volumes) e recria do zero
make clean && make up && make seed
```

---

## Testes automatizados

```bash
make test              # Unit + contract (não precisa do ambiente rodando)
make test-unit         # Só unit tests (Go + React/Vitest)
make test-integration  # Integration tests — requer make up rodando
make test-contract     # Validação de schemas JSON dos eventos
make test-frontend     # Só testes React (Vitest)
```

---

## Comandos

```bash
make up            # Sobe tudo do zero
make dev           # Sobe com hot reload (air para Go, Vite HMR para frontend)
make down          # Para todos os containers
make restart       # Restarta só os serviços (mantém LocalStack e dados)
make seed          # Popula inventário com prod-001, prod-002, prod-003
make logs          # Tail dos logs dos serviços Go
make logs-all      # Tail de todos os containers
make status        # Estado de saúde dos containers
make clean         # Remove tudo: containers + volumes + dados LocalStack
make infra-apply   # Re-provisiona infraestrutura Terraform
make infra-destroy # Destroi infraestrutura Terraform
make lint          # Roda linter em todos os serviços Go
make tidy          # go mod tidy em todos os serviços
```

---

## Estrutura do repositório

```
.
├── infra/                  # Terraform
│   ├── modules/            # Módulos reutilizáveis (sqs, sns, dynamodb, lambda, api_gateway, iam)
│   └── envs/dev/           # Ambiente dev (variáveis, main.tf)
├── services/               # Microserviços Go
│   ├── order-service/      # HTTP server (Chi router)
│   ├── payment-service/    # SQS consumer
│   ├── inventory-service/  # SQS consumer
│   └── notification-service/ # SQS consumer
├── frontend/               # React + TypeScript + Vite
├── docker/                 # Configs: Loki, Grafana, Promtail, air.toml
├── scripts/                # seed.sh, tf-apply.sh, test-*.sh
├── tests/                  # Integration e contract tests
└── docs/                   # ADRs e documentação técnica
```

---

## Produtos disponíveis (após make seed)

| ID | Nome | Preço |
|---|---|---|
| prod-001 | Mechanical Keyboard | R$ 299,99 |
| prod-002 | Wireless Mouse | R$ 79,99 |
| prod-003 | USB-C Hub | R$ 49,99 |
