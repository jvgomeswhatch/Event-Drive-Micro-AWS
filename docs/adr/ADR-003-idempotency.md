# ADR-003: Escrita condicional no DynamoDB para idempotência distribuída

**Status**: Aceito  
**Data**: 2025-01

## Contexto

SQS entrega mensagens **pelo menos uma vez**. Sem idempotência, uma entrega duplicada cobrariam o cliente duas vezes, reservariam estoque em dobro ou enviariam dois e-mails de confirmação. Cada etapa de processamento precisa ser segura para retry.

Alternativas:
1. **Set em memória**: rápido, perdido no restart, não funciona com múltiplas instâncias.
2. **Redis SET NX**: funciona, mas adiciona mais uma dependência.
3. **Escrita condicional no DynamoDB**: usa o DynamoDB já existente, atômico via `ConditionExpression: attribute_not_exists(idempotencyKey)`.

## Decisão

Usar uma tabela `idempotency-dev` compartilhada no DynamoDB com `PutItem` condicional. Cada serviço escreve uma chave `{servico}#{orderId}` antes de processar. Se a chave já existe, o DynamoDB lança `ConditionalCheckFailedException` — o processor pula com um log e retorna `nil` (sem retry).

## Design: claim → process → resolve

```
1. claim(key)         // PutItem com attribute_not_exists — atômico
2. process(event)     // lógica de negócio + escritas no DynamoDB
3. publish SNS        // notificação downstream
4. resolve(key)       // atualiza chave para "completed"
```

Se o serviço travar entre os passos 2 e 4, a chave permanece em estado `"processing"`. Na próxima entrega SQS, o `claim` retorna false e o evento é pulado. Essa é uma estratégia de **skip-on-duplicate**, não de **replay-on-failure** — adequada porque a ação de negócio (cobrança, decremento de estoque) já está persistida.

## Trade-offs

- **Skip vs replay**: se o processamento teve sucesso mas o publish SNS falhou, a chave de idempotência é resolvida antes de retornar o erro — garantindo que não haverá cobrança dupla. O caller recebe o erro e pode alertar/DLQ, mas um retry será pulado com segurança.
- **TTL**: chaves expiram depois de 24 horas, limpas automaticamente pelo TTL do DynamoDB. A retenção de mensagens SQS também é 24 horas por padrão.
- **Race condition**: duas entregas concorrentes da mesma mensagem tentam `PutItem` ao mesmo tempo. A escrita condicional do DynamoDB garante que exatamente uma vence; a outra recebe `ConditionalCheckFailedException`.

## Consequências

`idempotency-dev` é a única tabela compartilhada entre serviços. Cada serviço usa um prefixo de chave próprio para evitar colisões. A tabela tem um atributo `expiresAt` para TTL.
