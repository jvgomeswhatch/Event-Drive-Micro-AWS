# Security Review — Plataforma Event-Driven de Pedidos

> Revisão de segurança da plataforma serverless-first.
> Ambiente: local (LocalStack) com práticas espelhando produção real.

---

## 1. Superfície de Ataque

```
Internet
  └─ API Gateway (HTTP REST)
       └─ order-service (único ponto de entrada HTTP)
            ├─ SQS order-queue
            │    └─ payment-service (consumer)
            │         └─ SNS payment-events
            │              ├─ SQS payment-queue → inventory-service
            │              └─ SQS notification-queue → notification-service
            └─ DynamoDB (orders, idempotency)
```

Apenas o `order-service` expõe superfície HTTP. Os outros três serviços são consumers assíncronos — sua superfície de ataque é restrita à fronteira SQS/SNS, que não é acessível diretamente pela internet.

---

## 2. Autenticação e Autorização

### 2.1 JWT (order-service)

| Atributo       | Implementação                                                                 |
|----------------|-------------------------------------------------------------------------------|
| Algoritmo      | HS256 (HMAC-SHA256)                                                           |
| Secret         | Variável de ambiente `JWT_SECRET` — obrigatória em produção                   |
| Claims         | `sub` (customerId), `iss`, `exp`, `iat`                                       |
| Validação      | Assinatura + expiração verificadas em todo request protegido                  |
| Endpoint       | `POST /auth/token` — emite token para uso no dashboard/frontend               |

**Limitação conhecida:** HS256 com secret compartilhado é adequado para simulação local. Em produção, usar RS256 com par de chaves assimétricas para permitir verificação sem exposição do signing key.

**Rotas protegidas:**
- `POST /orders` — requer `Authorization: Bearer <token>`
- `GET /orders/:orderId` — requer `Authorization: Bearer <token>`
- `GET /orders?customerId=` — requer `Authorization: Bearer <token>`

**Rota pública:**
- `POST /auth/token` — emissão de token (sem autenticação prévia, por design)
- `GET /health` — health check (sem autenticação, necessário para orquestradores)

### 2.2 Autorização entre serviços

Os serviços downstream (payment, inventory, notification) não possuem endpoint HTTP exposto — comunicam-se exclusivamente via SQS/SNS. A "autorização" nesse nível é garantida pelas IAM policies (seção 3).

---

## 3. IAM — Princípio do Menor Privilégio

### 3.1 Estratégia

Cada serviço possui uma IAM role dedicada com policies específicas. Nenhum serviço tem acesso a recursos que não são de sua responsabilidade.

### 3.2 Matriz de permissões por serviço

| Serviço              | SQS (leitura)        | SQS (escrita)       | SNS (publicação)    | DynamoDB (tabelas)                          |
|----------------------|----------------------|---------------------|---------------------|---------------------------------------------|
| order-service        | order-queue          | order-queue, order-dlq | order-events      | orders, idempotency                         |
| payment-service      | payment-queue        | payment-dlq         | payment-events      | payments, idempotency, event-timeline       |
| inventory-service    | inventory-queue      | inventory-dlq       | inventory-events    | inventory, idempotency, event-timeline      |
| notification-service | notification-queue   | notification-dlq    | —                   | orders (UpdateItem), idempotency, event-timeline |

### 3.3 O que cada serviço NÃO pode fazer

- **order-service**: não pode ler de filas de outros serviços, não pode publicar em `payment-events` ou `inventory-events`.
- **payment-service**: não pode acessar a tabela `orders` ou `inventory`, não pode publicar em `order-events`.
- **inventory-service**: não pode acessar `payments`, não pode publicar em `payment-events`.
- **notification-service**: sem acesso a SNS (serviço terminal — apenas lê e atualiza status).

### 3.4 Limitação de simulação

No LocalStack, IAM é simulado sem enforcement real de políticas. As políticas definidas no Terraform documentam a intenção de segurança e seriam enforcement em AWS real.

---

## 4. Gestão de Secrets

| Item                  | Abordagem                                                                              |
|-----------------------|----------------------------------------------------------------------------------------|
| Secrets hardcoded     | Nenhum — `grep -r "password\|secret\|key" --include="*.go"` não retorna valores       |
| JWT_SECRET            | Variável de ambiente obrigatória; ausência retorna HTTP 500 (sem fallback)             |
| AWS credentials       | `test/test` apenas para LocalStack; em produção usar IAM roles (sem credenciais fixas) |
| `.env.example`        | Documenta todas as variáveis necessárias sem valores reais                             |
| `.gitignore`          | Arquivo `.env` ignorado por padrão                                                     |

**Nota:** `JWT_SECRET` é obrigatório — sem a variável definida, tanto a emissão (`/auth/token`) quanto a validação (middleware `Auth`) retornam HTTP 500. Recomendação para produção: usar AWS Secrets Manager e injetar na inicialização do container.

---

## 5. Validação de Input

### 5.1 Fronteira HTTP (order-service)

Aplicada em `internal/handler/order_handler.go` e `internal/security/sanitize.go`:

| Campo          | Validação                                                                   |
|----------------|-----------------------------------------------------------------------------|
| `customerId`   | Não vazio, max implícito pelo DynamoDB key size                             |
| `items`        | 1–50 itens; cada item com `productId` não vazio e `quantity` entre 1–1000  |
| `orderId` (GET)| Regex UUID v4 — bloqueia SQL injection e path traversal                    |
| `customerId` (query) | Não vazio                                                             |
| Corpo inválido | JSON decode com erro → 400 imediato                                        |

Funções de sanitização:
- `SanitizeString`: remove caracteres de controle não-printáveis
- `SanitizeID`: allowlist de caracteres (alfanumérico + `-` + `_`)
- `SafeLogString`: escapa `\n`/`\r` para prevenir log injection

### 5.2 Fronteira SQS/SNS (consumers)

Todos os consumers validam schema antes de deserializar:

```
raw bytes → eventschema.Validate(eventType, version, raw) → json.Unmarshal → domain struct
```

Validação falha → `ChangeMessageVisibility(0)` → roteamento imediato para DLQ (não-retentável).

Eventos de tipo desconhecido:
- `payment-service`: rejeita (schema obrigatório)
- `inventory-service`: aceita apenas `payment.succeeded`, descarta o resto
- `notification-service`: lista de allowlist explícita (`RelevantEvents`)

### 5.3 Injeção

| Vetor          | Mitigação                                                                   |
|----------------|-----------------------------------------------------------------------------|
| SQL Injection  | Não aplicável — DynamoDB usa expressões parametrizadas (SDK AWS)            |
| Log Injection  | `SafeLogString` escapa newlines antes de logar qualquer input externo       |
| Path Traversal | UUID regex em `orderId` bloqueia `../` e similares                          |
| NoSQL Injection| DynamoDB SDK usa `ExpressionAttributeValues` — sem concatenação de strings  |

---

## 6. Proteções de Transporte (HTTP)

Headers de segurança aplicados em `internal/security/headers.go` em todos os responses:

| Header                         | Valor                                      | Proteção                         |
|--------------------------------|--------------------------------------------|----------------------------------|
| `X-Content-Type-Options`       | `nosniff`                                  | MIME sniffing                    |
| `X-Frame-Options`              | `DENY`                                     | Clickjacking                     |
| `X-XSS-Protection`             | `1; mode=block`                            | XSS (browsers legados)           |
| `Strict-Transport-Security`    | `max-age=31536000; includeSubDomains`      | Downgrade HTTPS → HTTP           |
| `Referrer-Policy`              | `strict-origin-when-cross-origin`          | Vazamento de URL em referrer     |
| `Permissions-Policy`           | `geolocation=(), microphone=(), camera=()` | APIs sensíveis do browser        |
| `Server`                       | Removido                                   | Fingerprinting do servidor       |

### 6.1 CORS

Configurado em `internal/middleware/middleware.go`:

- Origens permitidas via `CORS_ALLOWED_ORIGINS` (env var, separado por vírgula)
- Em desenvolvimento (variável vazia): aceita qualquer origem
- Em produção: deve listar explicitamente as origens do frontend
- Header `Vary: Origin` adicionado quando há restrição — previne cache poisoning em proxies

---

## 7. Rate Limiting

Implementado em `internal/security/ratelimit.go`:

- Algoritmo: token bucket por IP com refil contínuo
- Configuração default: 10 req/s com burst de 20
- Extração de IP: respeita `X-Real-IP` e `X-Forwarded-For` (para proxies)
- Cleanup automático de buckets inativos (TTL de 5 minutos)
- Resposta em rate limit: `429 Too Many Requests` com log estruturado

**Limitação:** Rate limiting em memória não funciona em múltiplas instâncias do serviço. Em produção, usar Redis ou AWS API Gateway throttling para rate limiting distribuído.

---

## 8. Idempotência como Controle de Segurança

A idempotência não é apenas um requisito de consistência — é também um controle de segurança:

- Previne **replay attacks**: um request capturado e reenviado com o mesmo `X-Idempotency-Key` retorna o resultado cacheado sem reprocessar
- Previne **double-charging**: payment-service usa idempotência no `orderId` — mesmo evento processado duas vezes não gera dois pagamentos
- Implementação: DynamoDB `ConditionExpression: attribute_not_exists(idempotencyKey)` — atomicidade garantida pelo banco

---

## 9. Riscos Residuais e Recomendações para Produção

| Risco                           | Severidade | Mitigação Local           | Recomendação Produção                                  |
|---------------------------------|------------|---------------------------|--------------------------------------------------------|
| JWT secret compartilhado (HS256)| Médio      | Env var obrigatória       | Migrar para RS256 + rotação de chaves via Secrets Mgr  |
| CORS wildcard em dev            | Baixo      | Documentado, não padrão   | `CORS_ALLOWED_ORIGINS` obrigatório em produção         |
| Rate limiting em memória        | Médio      | Suficiente para 1 instância | Redis ou API Gateway throttling para multi-instância |
| IAM sem enforcement (LocalStack)| N/A (simulação) | Documentado          | Policies reais com AWS IAM + CloudTrail audit log      |
| Health endpoints sem auth       | Baixo      | Não exposto externamente  | Restringir a rede interna em produção                  |
| TLS não configurado localmente  | N/A (simulação) | HSTS header presente | TLS obrigatório em produção (ACM + ALB)                |

---

## 10. Checklist de Fase 5

- [x] IAM policies mínimas por serviço (least privilege) — roles e policies separadas por serviço
- [x] JWT auth simulation — HS256 com emissão e validação no order-service
- [x] Validação de input em todas as fronteiras — HTTP (order-service) e SQS/SNS (todos os consumers)
- [x] Sem secrets hardcoded — env vars + `.env.example` documentado
- [x] Proteção contra injection — sanitização de input, UUID regex, parametrização DynamoDB, log injection prevention
- [x] Security headers HTTP — `headers.go` aplicado em todos os responses
- [x] Rate limiting — token bucket por IP no order-service
- [x] CORS configurável — allowlist por env var, não wildcard em produção
- [x] Idempotência como controle de segurança
- [x] Security review documentado
