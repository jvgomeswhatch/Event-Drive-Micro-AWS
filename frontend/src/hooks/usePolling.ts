import { useEffect, useRef } from 'react'
import type { OrderStatus } from '@/types'
import { statusFinal } from '@/utils/status'

interface UsePollingOptions {
  orderId: string
  status: OrderStatus
  onAtualizar: (orderId: string) => Promise<void>
  intervalo?: number
  maxTentativas?: number
}

const DELAYS = [2000, 3000, 5000, 10000]
const MAX_TENTATIVAS_PADRAO = 30 // ~2 minutos no pior caso (30 × 10s)

// Faz polling de um pedido até atingir status final ou esgotar tentativas.
// Usa backoff progressivo: 2s → 3s → 5s → 10s → estabiliza em 10s.
export function usePolling({
  orderId,
  status,
  onAtualizar,
  intervalo = 2000,
  maxTentativas = MAX_TENTATIVAS_PADRAO,
}: UsePollingOptions) {
  const tentativasRef = useRef(0)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const onAtualizarRef = useRef(onAtualizar)

  useEffect(() => {
    onAtualizarRef.current = onAtualizar
  })

  useEffect(() => {
    if (statusFinal(status)) {
      tentativasRef.current = 0
      return
    }

    function agendarProximaPoll() {
      if (tentativasRef.current >= maxTentativas) return

      const delay = DELAYS[Math.min(tentativasRef.current, DELAYS.length - 1)] ?? intervalo

      timerRef.current = setTimeout(async () => {
        tentativasRef.current++
        try {
          await onAtualizarRef.current(orderId)
        } catch {
          // erro de rede não cancela o polling — próxima tentativa vai tentar novamente
        }
        agendarProximaPoll()
      }, delay)
    }

    agendarProximaPoll()

    return () => {
      if (timerRef.current) clearTimeout(timerRef.current)
    }
  }, [orderId, status, intervalo, maxTentativas])
}
