# ADR-004: LocalStack para simulação local da AWS

**Status**: Aceito  
**Data**: 2025-01

## Contexto

A plataforma depende de SQS, SNS, DynamoDB, Lambda, IAM, S3 e API Gateway. Rodar contra AWS real durante o desenvolvimento tem dois problemas: custo (cada máquina de dev incorre em cobranças AWS) e fricção (exige credenciais AWS, políticas IAM, recursos por região).

Alternativas:
1. **Mockar tudo no código** (fakes em memória): rápido, mas diverge do comportamento real (escritas condicionais DynamoDB, visibility timeout SQS, fan-out SNS).
2. **Ambiente AWS real de dev**: preciso, mas caro, exige rede, estado compartilhado entre devs.
3. **LocalStack**: container Docker que emula as APIs AWS em `localhost:4566`.

## Decisão

Usar LocalStack 3.4 Community Edition como única dependência AWS para desenvolvimento local.

## O que o LocalStack cobre bem

- DynamoDB CRUD, escritas condicionais, queries com GSI, TTL
- SQS: send, receive, delete, visibility timeout, redrive para DLQ
- SNS: publish, fan-out de subscription para SQS, message attributes
- S3: para simular remote state do Terraform
- IAM: avaliação de policies (parcial — veja limitações)

## Limitações conhecidas vs AWS real

| Área | Comportamento LocalStack | AWS real |
|---|---|---|
| Avaliação de policies IAM | Permissivo — a maioria das chamadas funciona independente da policy | Enforcement rigoroso |
| Cold start Lambda | Quase instantâneo | 100 ms – 10 s dependendo do runtime |
| SQS FIFO | Suportado mas não testado neste projeto | Garantias de ordenação completas |
| DynamoDB Streams | Não usado neste projeto | Disponível |
| CloudWatch Logs | Parcial | Ingestão de logs + alertas completos |
| SNS email/SMS | Sem entrega real | Entrega efetiva |
| API Gateway v2 (HTTP) | Limitado | Completo |

## Consequências

O Terraform aplica contra `http://localstack:4566` usando credenciais fictícias (`test`/`test`). O AWS SDK é configurado com override de `BaseEndpoint`. Todos os ARNs de recursos usam a conta `000000000000` (padrão do LocalStack). ARNs estão no `.env` para dev local (`arn:aws:sns:us-east-1:000000000000:...`).

Antes de fazer deploy na AWS real, as policies IAM precisam ser validadas contra a AWS de verdade — o IAM permissivo do LocalStack é uma lacuna significativa.
