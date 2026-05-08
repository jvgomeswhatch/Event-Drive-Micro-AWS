# Cenários de Falha

Documenta como a plataforma se comporta sob diversas condições de falha e quais são os sintomas observáveis.

---

## 1. Payment-service trava no meio do processamento

**Gatilho**: container do payment-service morre depois de persistir o registro de pagamento mas antes de publicar no SNS.

**Comportamento**:
- Visibility timeout do SQS expira → mensagem é reentregue
- No retry: chave de idempotência existe como `"processing"` → `claim()` retorna false → mensagem pulada
- Resultado: pagamento persistido no DynamoDB, mas evento SNS nunca publicado → inventory/notification nunca acionados
- Pedido fica em estado `processing` indefinidamente

**Detecção**: query no Grafana Loki `{service="payment-service"} |= "Duplicate payment event"` sem entrada correspondente de `"Payment processed"`.

**Mitigação**: O caminho de erro do publish SNS (se o serviço trava *durante* o publish, não antes) resolve a chave de idempotência primeiro para evitar cobrança dupla. A notificação perdida é uma perda de dados aceitável nesta simulação; um sistema de produção usaria o padrão transactional outbox.

---

## 2. Falha no publish SNS (rede ou restart do LocalStack)

**Gatilho**: endpoint SNS temporariamente indisponível quando o payment-service tenta publicar `payment.succeeded`.

**Comportamento**:
- `sns.Publish` retorna erro
- Chave de idempotência é resolvida (pagamento já persistido — sem cobrança dupla no retry)
- `Process()` retorna erro → loop de retry do consumer SQS ativa
- Exponential backoff: tentativa 1 → 2 → 3, depois mensagem vai para a DLQ `payment-dlq-dev`
- Serviços downstream (inventory, notification) nunca recebem o evento

**Detecção**: `docker logs payment-service | grep "publish payment event to SNS"` + contagem de mensagens na DLQ aumentando.

**Recuperação**: Corrigir o SNS, depois redriving as mensagens da DLQ via console AWS ou:
```bash
aws --endpoint-url=http://localhost:4566 --region=us-east-1 \
  sqs send-message --queue-url http://localhost:4566/000000000000/payment-queue-dev \
  --message-body "$(aws --endpoint-url=http://localhost:4566 sqs receive-message \
    --queue-url http://localhost:4566/000000000000/payment-dlq-dev \
    --query 'Messages[0].Body' --output text)"
```

---

## 3. Estoque insuficiente

**Gatilho**: pedido contém um `productId` que não está na tabela de inventário, ou quantidade excede o estoque disponível.

**Comportamento**:
- `ConditionExpression: quantity >= :qty AND attribute_exists(productId)` do DynamoDB `UpdateItem` falha
- Processor define `status="failed"`, faz rollback de itens já reservados parcialmente
- Publica `inventory.failed` no `inventory-events-dev`
- notification-service recebe o evento e atualiza o status do pedido para `failed`
- Frontend exibe estado "failed" com o motivo da falha

**Detecção**: timeline de eventos do pedido mostra entrada `inventory.failed`.

---

## 4. Entrega duplicada de mensagem SQS

**Gatilho**: SQS entrega a mesma mensagem `order.created` duas vezes (garantia at-least-once).

**Comportamento**:
- payment-service recebe a mensagem, reivindica chave de idempotência `payment-service#<orderId>`
- Segunda entrega chega: `claim()` bate em `ConditionalCheckFailedException` → retorna `false`
- Log: `"Duplicate payment event — skipping"`
- Sem cobrança dupla, sem segunda escrita no DynamoDB

**Verificação**:
```bash
# Enviar com a mesma chave de idempotência duas vezes
curl -X POST http://localhost:3001/orders \
  -H "X-Idempotency-Key: test-idem-key" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"customerId":"user-123","items":[{"productId":"prod-001","quantity":1,"unitPrice":299.99}]}'

curl -X POST http://localhost:3001/orders \
  -H "X-Idempotency-Key: test-idem-key" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"customerId":"user-123","items":[{"productId":"prod-001","quantity":1,"unitPrice":299.99}]}'
# Segunda resposta: {"message":"duplicate request","cached":true}
```

---

## 5. Falha de pagamento (simulada)

**Gatilho**: `simulateFailure: true` no body do request de criação de pedido.

**Comportamento**:
- payment-service define `status="failed"`, persiste o registro de pagamento
- Publica `payment.failed` no SNS
- inventory-service não assina `payment.failed` (apenas `payment.succeeded`) — nenhum estoque reservado
- notification-service assina os dois → atualiza status do pedido para `failed`

**Teste**:
```bash
curl -X POST http://localhost:3001/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"customerId":"user-123","items":[{"productId":"prod-001","quantity":1,"unitPrice":299.99}],"simulateFailure":true}'
```

---

## 6. Restart do LocalStack com volumes persistentes

**Gatilho**: `docker compose restart localstack`

**Comportamento**:
- `PERSISTENCE=1` no docker-compose.yml → LocalStack salva estado no volume `localstack-data`
- Filas, tópicos, tabelas e seus conteúdos sobrevivem ao restart
- Serviços reconectam automaticamente (usam `retry` com exponential backoff na inicialização)

**Reset limpo**:
```bash
make clean && make up && make seed
```

---

## 7. Acúmulo de mensagens na DLQ

**Gatilho**: Uma mensagem falha em todas as 3 tentativas de retry.

**Comportamento**:
- Mensagem movida para a DLQ daquele serviço (ex: `payment-dlq-dev`)
- Serviço continua processando outras mensagens (DLQ é separada da fila principal)
- Pedido associado à mensagem que falhou fica em estado `processing`

**Inspecionar DLQ**:
```bash
aws --endpoint-url=http://localhost:4566 --region=us-east-1 \
  sqs receive-message --queue-url http://localhost:4566/000000000000/payment-dlq-dev \
  --attribute-names All
```

**Frontend**: o visualizador de DLQ no dashboard mostra a contagem de mensagens por fila.
