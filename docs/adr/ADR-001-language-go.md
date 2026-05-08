# ADR-001: Go como linguagem de backend para todos os microserviços

**Status**: Aceito  
**Data**: 2025-01

## Contexto

A plataforma precisa de quatro microserviços: um entry point HTTP (order-service) e três consumers SQS (payment, inventory, notification). A escolha da linguagem impacta tamanho do binário, tempo de startup do container, modelo de concorrência e overhead operacional.

Alternativas consideradas: Python (FastAPI/Celery), Node.js (Express), Go.

## Decisão

Usar Go 1.24 para todos os quatro serviços.

## Justificativa

- **Binário único por serviço**: sem runtime, sem dependency hell no startup do container. Imagem Alpine fica em ~15 MB vs 200+ MB com Node/Python.
- **Goroutines para polling SQS**: o loop de long-polling mapeia naturalmente para goroutines com cancelamento via `context.Context`. Sem configuração de thread pool ou cerimônia de async/await.
- **Startup rápido**: crítico para targets Lambda e para containers que fazem cold-start frequente no dev local.
- **Biblioteca padrão cobre a maior parte**: `net/http`, `encoding/json`, `context`, `sync` — sem framework pesado.
- **Tratamento de erros explícito**: o tipo de retorno `error` força que todo caminho de falha seja tratado. Importante em sistemas event-driven onde erros silenciosos causam inconsistência de dados.

## Trade-offs

- **Mais verboso**: sem roteamento por decorator, sem herança de classe. Boilerplate de handlers HTTP e injeção de dependência é manual.
- **Ecossistema menor** que Node/Python para algumas integrações AWS. Mitigado pelo `aws-sdk-go-v2` oficial.
- **Curva de aprendizado** para quem vem de linguagens dinâmicas: tipos explícitos, generics não é first-class na stdlib (Go < 1.18).

## Consequências

Todos os serviços usam o mesmo padrão de `Dockerfile` (multi-stage build: `golang:1.24-alpine` builder → `alpine:3.20` final). Vendor mode (`-mod=vendor`) garante builds offline no CI e nos containers sem internet.
