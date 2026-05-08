# ADR-002: Fan-out via SNS em vez de publicação direta no SQS

**Status**: Aceito  
**Data**: 2025-01

## Contexto

Quando o order-service cria um pedido, os serviços downstream (payment, inventory, notification) precisam receber os eventos. Duas topologias foram consideradas:

1. **SQS direto**: order-service escreve em cada fila downstream diretamente. Simples, mas cria acoplamento — order-service precisa conhecer cada consumer.
2. **Fan-out via SNS**: order-service publica num tópico SNS; o SNS entrega para cada fila SQS inscrita. Desacoplado — adicionar um novo consumer é uma mudança no Terraform, não no código.

## Decisão

Usar fan-out via SNS para toda propagação de eventos entre serviços:
- `order-service` → SNS `order-events-dev` → SQS `payment-queue-dev`
- `payment-service` → SNS `payment-events-dev` → SQS `inventory-queue-dev`, `notification-queue-dev`
- `inventory-service` → SNS `inventory-events-dev` → SQS `notification-queue-dev`

## Justificativa

- **Princípio aberto/fechado na infraestrutura**: novos consumers se inscrevem num tópico existente sem tocar no código do produtor.
- **Filtragem de mensagens**: filter policies de subscription SNS permitem que consumers recebam apenas os tipos de evento relevantes (ex: notification-service só precisa de `payment.*` e `inventory.*`).
- **Log de auditoria**: o SNS pode fazer fan-out para múltiplos destinos simultaneamente (SQS + CloudWatch Logs + Lambda) sem mudanças no produtor.

## Trade-offs

- **Hop extra**: SNS adiciona ~10 ms de latência vs escrita direta no SQS. Irrelevante para fluxos assíncronos.
- **Mais infraestrutura**: N tópicos + M subscriptions vs N filas. Os módulos Terraform gerenciam isso.
- **Debugging**: uma mensagem percorre SNS → SQS → consumer. O Correlation ID propagado via `MessageAttributes` viabiliza o rastreamento distribuído.

## Consequências

Cada serviço publica no seu próprio tópico SNS. As variáveis de ambiente `PAYMENT_EVENTS_TOPIC_ARN`, `INVENTORY_EVENTS_TOPIC_ARN`, `ORDER_EVENTS_TOPIC_ARN` carregam o ARN em runtime. Dead-letter queues ficam acopladas às filas SQS (não aos tópicos SNS) para capturar falhas de processamento.
