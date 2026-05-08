# Estratégia de Compatibilidade de Schemas de Eventos

## Modelo de versionamento

Cada evento carrega campos explícitos `version` e `eventType`:

```json
{ "eventType": "order.created", "version": "1", ... }
```

Os arquivos de schema ficam em `docs/event-schemas/v{N}/` e são incorporados em cada serviço via o pacote `internal/eventschema` em tempo de compilação.

---

## Regras para mudanças retrocompatíveis (permitidas sem bump de versão)

Estas mudanças podem ser feitas em um schema `v1` existente sem quebrar os consumers:

| Mudança | Seguro? | Motivo |
|--------|-------|--------|
| Adicionar campo opcional com valor padrão | ✅ | Consumers ignoram campos desconhecidos |
| Ampliar restrição de string (`maxLength`) | ✅ | Valores existentes continuam válidos |
| Adicionar novo valor de enum permitido | ✅ | Consumers devem tratar valores desconhecidos graciosamente |
| Adicionar novo objeto aninhado opcional | ✅ | Consumer ignora chaves desconhecidas |

## Regras para mudanças incompatíveis (exigem bump de versão)

| Mudança | Incompatível? | Ação |
|--------|-----------|--------|
| Remover campo obrigatório | ❌ | Bump para `v2`, executar ambos em paralelo |
| Renomear campo | ❌ | Bump para `v2` |
| Alterar tipo de campo | ❌ | Bump para `v2` |
| Alterar valor `const` do `eventType` | ❌ | Novo tipo de evento |
| Tornar campo opcional em obrigatório | ❌ | Bump para `v2` |
| Restringir constraint (ex: reduzir `maxLength`) | ❌ | Bump para `v2` |

---

## Guia de migração de versão

Quando uma mudança incompatível for necessária:

### 1. Publicar ambas as versões em paralelo

O produtor publica **v1 e v2 simultaneamente** durante a migração:

```
produtor → publica order.created@v1  (consumers legados)
         → publica order.created@v2  (novos consumers)
```

Na prática: publicar apenas `v2` e usar um adaptador de schema no nível do consumer para aceitar ambos até que todos os consumers sejam atualizados.

### 2. Caminho de atualização dos consumers

```
1. Fazer deploy do novo consumer que aceita v1 E v2
2. Fazer deploy do produtor que publica v2
3. Aguardar todas as mensagens v1 drenarem da fila
4. Remover o tratamento de v1 do consumer
5. Arquivar o arquivo de schema v1 (nunca deletar — manter para auditoria)
```

### 3. Atualização de filter policy

As filter policies de subscription do SNS devem ser atualizadas para incluir a nova versão caso filtrem pelo atributo `version`.

---

## Regras dos consumers (aplicadas em código)

1. **eventType desconhecido** → aceitar e descartar (compatibilidade futura)
2. **Versão desconhecida** → rejeitar para DLQ (consumer precisa ser atualizado)
3. **Tipo e versão conhecidos, mas campo obrigatório ausente** → rejeitar para DLQ
4. **Validação ocorre antes do unmarshalling** — evita panics por estado parcial

---

## Changelog de schemas

| Versão | Evento | Data | Mudança |
|---------|-------|------|--------|
| v1 | order.created | 2026-05-05 | Schema inicial |
| v1 | payment.succeeded | 2026-05-05 | Schema inicial |
| v1 | payment.failed | 2026-05-05 | Schema inicial |
| v1 | inventory.reserved | 2026-05-05 | Schema inicial |
| v1 | inventory.failed | 2026-05-05 | Schema inicial |
