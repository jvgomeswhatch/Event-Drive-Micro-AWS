export function gerarCorrelationId(): string {
  return crypto.randomUUID()
}
