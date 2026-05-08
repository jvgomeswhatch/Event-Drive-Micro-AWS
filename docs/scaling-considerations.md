# Considerações de Escalabilidade

Como a arquitetura escalaria do dev local para carga de produção, e o que precisaria mudar.

---

## Setup atual (dev local)

| Componente | Instâncias | Concorrência |
|---|---|---|
| order-service | 1 | 10 req/s (rate limiter) |
| payment-service | 1 | 1 mensagem SQS por vez |
| inventory-service | 1 | 1 mensagem SQS por vez |
| notification-service | 1 | 1 mensagem SQS por vez |
| DynamoDB (LocalStack) | 1 | Nó único |

---

## Escalando consumers SQS

Cada consumer faz poll do SQS em uma única goroutine. Para aumentar o throughput:

```go
// Atual: sequencial
for {
    msgs, _ := sqs.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{MaxNumberOfMessages: 10})
    for _, msg := range msgs { process(msg) } // sequencial
}

// Escalado: paralelo com concorrência limitada
sem := make(chan struct{}, workers) // ex: workers=20
for _, msg := range msgs {
    sem <- struct{}{}
    go func(m types.Message) {
        defer func() { <-sem }()
        process(m)
    }(msg)
}
```

Na AWS real, Lambda com SQS event source mapping escala automaticamente: a AWS gerencia invocações concorrentes de Lambda baseado na profundidade da fila. O módulo `infra/modules/lambda` provisiona essa infraestrutura.

---

## Capacidade do DynamoDB

O LocalStack ignora unidades de leitura/escrita. Na AWS real:

- `orders-dev`: alta escrita (criação de pedido + atualizações de status). Use modo de cobrança **on-demand** inicialmente.
- `idempotency-dev`: alto throughput de escrita com TTL. Considere capacidade provisionada com auto-scaling.
- `inventory-dev`: alta contenção em chaves `productId` populares. Updates condicionais do DynamoDB são atômicos, mas partições quentes podem causar throttling. Mitigação: write sharding ou cache DAX para leituras intensas.

---

## Escalabilidade horizontal do order-service

O order-service é stateless (auth JWT, sem estado de sessão em memória). Escala horizontalmente:
1. Rodando múltiplas instâncias atrás de um load balancer (ALB).
2. O rate limiter é por instância (token bucket em memória). Para rate limiting compartilhado, substituir por Redis INCR + TTL.

---

## Correlation ID e tracing em escala

O `correlationId` atual é logado em JSON mas não exportado para um backend de tracing. Em escala:
- Adicionar OpenTelemetry SDK: `go.opentelemetry.io/otel`
- Exportar spans para Jaeger ou AWS X-Ray
- Propagar header W3C `traceparent` em vez de `X-Correlation-ID`

---

## O que NÃO precisaria mudar

- **Versionamento de schema de eventos**: schemas são versionados (`v1/`) e validados na entrada do consumer. Adicionar `v2` sem quebrar subscribers de `v1` já está documentado em `docs/event-schemas/COMPATIBILITY.md`.
- **Padrão DLQ**: já acoplado a todas as filas SQS. Escalar não muda o limite de isolamento de falha.
- **Idempotência**: a escrita condicional do DynamoDB é atômica e funciona corretamente com N instâncias concorrentes do mesmo serviço.
- **Módulos Terraform**: parametrizados por ambiente (`tfvars`). Adicionar um ambiente `staging` ou `production` é um novo arquivo `tfvars`.

---

## Lambda vs containers de longa duração

A plataforma roda os serviços Go como containers Docker de longa duração (fazendo poll SQS em loop). Os stubs Lambda em `infra/modules/lambda` representam a alternativa serverless:

| Abordagem | Cold start | Modelo de custo | Overhead operacional |
|---|---|---|---|
| Container de longa duração (atual) | Nenhum | EC2/ECS sempre ligado | Atualizar + reiniciar no deploy |
| Lambda (alternativa) | 100 ms – 2 s (Go: < 500 ms) | Paga por invocação | Gerenciado pela AWS, escala até zero |

Para processamento SQS event-driven, Lambda com SQS event source mapping é a escolha natural em produção. A abordagem com containers é mais simples para dev local.
