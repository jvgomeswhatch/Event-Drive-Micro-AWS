import { useCallback } from 'react'
import { useAuthStore } from '@/store/auth'
import { useOrdersStore } from '@/store/orders'
import { criarPedido, buscarPedido, listarPedidos, buscarTimeline } from '@/api/orders'
import type { CreateOrderPayload } from '@/types'
import { gerarCorrelationId } from '@/utils/correlationId'
import { ApiError } from '@/api/client'

export function usePedidos() {
  const token = useAuthStore((s) => s.token)
  const customerId = useAuthStore((s) => s.customerId)

  const setPedidos = useOrdersStore((s) => s.setPedidos)
  const setStatusLista = useOrdersStore((s) => s.setStatusLista)
  const atualizarPedido = useOrdersStore((s) => s.atualizarPedido)
  const setTimeline = useOrdersStore((s) => s.setTimeline)
  const setStatusDetalhe = useOrdersStore((s) => s.setStatusDetalhe)
  const setStatusCriacao = useOrdersStore((s) => s.setStatusCriacao)
  const setErroCriacao = useOrdersStore((s) => s.setErroCriacao)

  const carregar = useCallback(async () => {
    if (!token || !customerId) return
    setStatusLista('loading')
    try {
      const pedidos = await listarPedidos(customerId, token)
      setPedidos(pedidos)
      setStatusLista('success')
    } catch {
      setStatusLista('error')
    }
  }, [token, customerId, setPedidos, setStatusLista])

  const atualizar = useCallback(
    async (orderId: string) => {
      if (!token) return
      try {
        const pedido = await buscarPedido(orderId, token)
        atualizarPedido(pedido)
      } catch (err) {
        console.debug('[polling] falha ao atualizar pedido', orderId, err)
      }
    },
    [token, atualizarPedido],
  )

  const carregarTimeline = useCallback(
    async (orderId: string) => {
      if (!token) return
      setStatusDetalhe('loading')
      try {
        const eventos = await buscarTimeline(orderId, token)
        setTimeline(eventos)
        setStatusDetalhe('success')
      } catch {
        setStatusDetalhe('error')
      }
    },
    [token, setTimeline, setStatusDetalhe],
  )

  const criar = useCallback(
    async (payload: CreateOrderPayload): Promise<string | null> => {
      if (!token) return null
      setStatusCriacao('loading')
      setErroCriacao(null)
      try {
        const idempotencyKey = gerarCorrelationId()
        const res = await criarPedido(payload, token, idempotencyKey)
        atualizarPedido({
          orderId: res.orderId,
          customerId: payload.customerId,
          items: payload.items,
          status: 'pending',
          correlationId: res.correlationId,
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
          simulateFailure: payload.simulateFailure,
        })
        setStatusCriacao('success')
        return res.orderId
      } catch (err) {
        const msg = err instanceof ApiError ? err.message : 'Erro inesperado ao criar pedido'
        setErroCriacao(msg)
        setStatusCriacao('error')
        return null
      }
    },
    [token, atualizarPedido, setStatusCriacao, setErroCriacao],
  )

  return { carregar, atualizar, carregarTimeline, criar }
}
