import { apiClient } from './client'
import type { Order, CreateOrderPayload, TimelineEvent } from '@/types'
import { gerarCorrelationId } from '@/utils/correlationId'

interface CriarPedidoResponse {
  orderId: string
  status: string
  correlationId: string
}

interface ListarPedidosResponse {
  orders: Order[]
}

export async function criarPedido(
  payload: CreateOrderPayload,
  token: string,
  idempotencyKey: string,
): Promise<CriarPedidoResponse> {
  return apiClient.post<CriarPedidoResponse>('/orders', payload, {
    token,
    correlationId: gerarCorrelationId(),
    idempotencyKey,
  })
}

export async function buscarPedido(orderId: string, token: string): Promise<Order> {
  return apiClient.get<Order>(`/orders/${orderId}`, { token })
}

export async function listarPedidos(customerId: string, token: string): Promise<Order[]> {
  const res = await apiClient.get<ListarPedidosResponse>(`/orders?customerId=${encodeURIComponent(customerId)}`, {
    token,
  })
  return res.orders
}

export async function buscarTimeline(orderId: string, token: string): Promise<TimelineEvent[]> {
  const res = await apiClient.get<{ events: TimelineEvent[] }>(`/orders/${orderId}/timeline`, { token })
  return res.events ?? []
}
