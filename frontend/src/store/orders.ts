import { create } from 'zustand'
import type { Order, TimelineEvent, AsyncStatus } from '@/types'

interface OrdersStore {
  pedidos: Order[]
  pedidoSelecionado: Order | null
  timeline: TimelineEvent[]
  statusLista: AsyncStatus
  statusDetalhe: AsyncStatus
  statusCriacao: AsyncStatus
  erroCriacao: string | null
  setPedidos: (pedidos: Order[]) => void
  atualizarPedido: (pedido: Order) => void
  selecionarPedido: (pedido: Order | null) => void
  setTimeline: (eventos: TimelineEvent[]) => void
  setStatusLista: (s: AsyncStatus) => void
  setStatusDetalhe: (s: AsyncStatus) => void
  setStatusCriacao: (s: AsyncStatus) => void
  setErroCriacao: (err: string | null) => void
}

export const useOrdersStore = create<OrdersStore>((set) => ({
  pedidos: [],
  pedidoSelecionado: null,
  timeline: [],
  statusLista: 'idle',
  statusDetalhe: 'idle',
  statusCriacao: 'idle',
  erroCriacao: null,

  setPedidos: (pedidos) => set({ pedidos }),

  atualizarPedido: (pedido) =>
    set((state) => {
      const existe = state.pedidos.find((p) => p.orderId === pedido.orderId)
      const pedidos = existe
        ? state.pedidos.map((p) => (p.orderId === pedido.orderId ? pedido : p))
        : [pedido, ...state.pedidos]
      const pedidoSelecionado =
        state.pedidoSelecionado?.orderId === pedido.orderId ? pedido : state.pedidoSelecionado
      return { pedidos, pedidoSelecionado }
    }),

  selecionarPedido: (pedido) => set({ pedidoSelecionado: pedido }),
  setTimeline: (timeline) => set({ timeline }),
  setStatusLista: (statusLista) => set({ statusLista }),
  setStatusDetalhe: (statusDetalhe) => set({ statusDetalhe }),
  setStatusCriacao: (statusCriacao) => set({ statusCriacao }),
  setErroCriacao: (erroCriacao) => set({ erroCriacao }),
}))
