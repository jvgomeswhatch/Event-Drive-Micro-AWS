export function formatarData(iso: string): string {
  try {
    return new Intl.DateTimeFormat('pt-BR', {
      dateStyle: 'short',
      timeStyle: 'medium',
    }).format(new Date(iso))
  } catch {
    return iso
  }
}

export function formatarServico(service: string): string {
  const mapa: Record<string, string> = {
    'order-service': 'Serviço de Pedidos',
    'payment-service': 'Serviço de Pagamento',
    'inventory-service': 'Serviço de Estoque',
    'notification-service': 'Serviço de Notificação',
  }
  return mapa[service] ?? service
}
