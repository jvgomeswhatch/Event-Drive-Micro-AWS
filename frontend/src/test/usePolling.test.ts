import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { usePolling } from '@/hooks/usePolling'

describe('usePolling', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('não executa polling quando status é final', async () => {
    const onAtualizar = vi.fn().mockResolvedValue(undefined)
    renderHook(() =>
      usePolling({ orderId: 'abc', status: 'confirmed', onAtualizar }),
    )
    await vi.advanceTimersByTimeAsync(10000)
    expect(onAtualizar).not.toHaveBeenCalled()
  })

  it('executa polling quando status não é final', async () => {
    const onAtualizar = vi.fn().mockResolvedValue(undefined)
    renderHook(() =>
      usePolling({ orderId: 'abc', status: 'pending', onAtualizar }),
    )
    await vi.advanceTimersByTimeAsync(2500)
    expect(onAtualizar).toHaveBeenCalledWith('abc')
  })

  it('para o polling quando componente é desmontado', async () => {
    const onAtualizar = vi.fn().mockResolvedValue(undefined)
    const { unmount } = renderHook(() =>
      usePolling({ orderId: 'abc', status: 'pending', onAtualizar }),
    )
    unmount()
    await vi.advanceTimersByTimeAsync(10000)
    expect(onAtualizar).not.toHaveBeenCalled()
  })

  // Bug fix: polling infinito quando API retorna erro repetidamente
  //
  // Bug anterior: onAtualizar lançava exceção → agendarProximaPoll() nunca
  // verificava se estava no limite → polling continuava infinitamente.
  // Com muitas abas abertas ou API caída, vira DoS acidental contra o servidor.
  // Correção: maxTentativas limita o total de chamadas; erros são capturados
  // dentro do setTimeout sem cancelar o ciclo.

  it('para após maxTentativas quando API falha repetidamente', async () => {
    const onAtualizar = vi.fn().mockRejectedValue(new Error('network error'))
    renderHook(() =>
      usePolling({ orderId: 'abc', status: 'pending', onAtualizar, maxTentativas: 3 }),
    )
    // Avança tempo suficiente para esgotar as 3 tentativas (2s + 3s + 5s = 10s)
    await vi.advanceTimersByTimeAsync(15000)
    expect(onAtualizar).toHaveBeenCalledTimes(3)
  })

  it('respeita maxTentativas padrão de 30', async () => {
    const onAtualizar = vi.fn().mockResolvedValue(undefined)
    renderHook(() =>
      usePolling({ orderId: 'abc', status: 'pending', onAtualizar }),
    )
    // Avança tempo além de 30 tentativas (30 × 10s = 300s)
    await vi.advanceTimersByTimeAsync(400000)
    expect(onAtualizar).toHaveBeenCalledTimes(30)
  })

  it('não conta tentativas para status final', async () => {
    const onAtualizar = vi.fn().mockResolvedValue(undefined)
    renderHook(() =>
      usePolling({ orderId: 'abc', status: 'confirmed', onAtualizar, maxTentativas: 3 }),
    )
    await vi.advanceTimersByTimeAsync(15000)
    expect(onAtualizar).not.toHaveBeenCalled()
  })
})
