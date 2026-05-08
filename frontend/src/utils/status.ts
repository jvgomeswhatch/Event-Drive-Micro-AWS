import type { OrderStatus } from '@/types'

interface StatusInfo {
  rotulo: string
  cor: string
  descricao: string
}

const mapa: Record<OrderStatus, StatusInfo> = {
  pending: {
    rotulo: 'Pendente',
    cor: 'warning',
    descricao: 'Pedido criado, aguardando processamento de pagamento.',
  },
  payment_confirmed: {
    rotulo: 'Pagamento confirmado',
    cor: 'info',
    descricao: 'Pagamento aprovado. Aguardando reserva de estoque.',
  },
  payment_failed: {
    rotulo: 'Pagamento falhou',
    cor: 'error',
    descricao: 'O pagamento foi recusado. Verifique os dados e tente novamente.',
  },
  confirmed: {
    rotulo: 'Confirmado',
    cor: 'success',
    descricao: 'Pedido confirmado. Itens reservados no estoque.',
  },
  fulfillment_failed: {
    rotulo: 'Sem estoque',
    cor: 'error',
    descricao: 'Não foi possível reservar os itens. Estoque insuficiente.',
  },
}

export function infoStatus(status: OrderStatus): StatusInfo {
  return mapa[status] ?? { rotulo: status, cor: 'info', descricao: '' }
}

export function statusFinal(status: OrderStatus): boolean {
  return ['payment_failed', 'confirmed', 'fulfillment_failed'].includes(status)
}
