# ADR-005: Gerenciamento de estado no frontend — Zustand + polling

**Status**: Aceito  
**Data**: 2025-01

## Contexto

O dashboard de pedidos precisa exibir atualizações de status em tempo real. Pedidos passam por múltiplos estados assíncronos (`pending` → `processing` → `completed`/`failed`), cada um atualizado por um microserviço diferente ao longo de alguns segundos. O frontend precisa refletir essas transições sem recarregar a página manualmente.

Alternativas para atualizações em tempo real:
1. **WebSockets**: bidirecional, baixa latência. Exige servidor WebSocket ou API Gateway WebSocket (infraestrutura significativa).
2. **Server-Sent Events (SSE)**: mais simples que WebSocket, só server-push. Ainda exige HTTP/2 ou conexão persistente.
3. **Polling com backoff progressivo**: mais simples, funciona com qualquer servidor HTTP, degrada bem.

Alternativas para estado do cliente:
1. **React Context + useReducer**: built-in, bom para apps pequenos, sem devtools.
2. **Redux Toolkit**: completo, pesado, overkill para 4 telas.
3. **Zustand**: mínimo (< 1 KB), suporte a devtools, sem boilerplate de Provider.

## Decisão

- **Polling** com backoff progressivo: poll imediato na criação do pedido, depois intervalos de 2s → 3s → 5s → 10s (estabiliza em 10s) até estado terminal ou máximo de 30 tentativas (~2 minutos).
- **Zustand** para estado global: token de autenticação, lista de pedidos atual, correlationId ativo.

## Justificativa

Polling é adequado porque:
- Pedidos chegam ao estado terminal em 5–10 segundos no caminho feliz.
- O backend é HTTP puro (sem suporte a WebSocket).
- Backoff progressivo reduz carga no servidor após o burst inicial.

Zustand em vez de Context:
- Zero boilerplate para actions (mutação direta de estado via setters estilo Immer).
- Funciona fora de componentes React (útil para interceptors do `apiClient`).

## Trade-offs

- **Polling cria carga N×RPS**: com 10 usuários simultâneos fazendo poll 6 vezes em 30 segundos = 60 requests extras. Aceitável nesta escala; precisaria de SSE ou WebSockets para 1000+ usuários simultâneos.
- **Estado do Zustand é persistido em `localStorage`** via `zustand/persist` (chave `platform-auth`). Token sobrevive a refresh, mas invalida após restart dos containers se `JWT_SECRET` mudar. Em produção, usar cookies HttpOnly para evitar acesso via XSS.
- **Sem otimistic updates**: a UI mostra o estado do servidor, não uma mutação otimista. Aceitável para fluxos assíncronos onde o servidor é a fonte da verdade.

## Consequências

O hook `usePolling` encapsula a lógica de backoff e é testado em isolamento. O intervalo de polling é configurável por componente. O frontend não usa TanStack Query (apesar de ter estado num rascunho inicial) — Zustand + hooks customizados são primitivas suficientes de data-fetching para este caso de uso.
