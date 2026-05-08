import { describe, it, expect } from 'vitest'
import { infoStatus, statusFinal } from '@/utils/status'
import type { OrderStatus } from '@/types'

describe('infoStatus', () => {
  const casos: [OrderStatus, string][] = [
    ['pending', 'Pendente'],
    ['payment_confirmed', 'Pagamento confirmado'],
    ['payment_failed', 'Pagamento falhou'],
    ['confirmed', 'Confirmado'],
    ['fulfillment_failed', 'Sem estoque'],
  ]

  it.each(casos)('status "%s" retorna rótulo "%s"', (status, rotulo) => {
    expect(infoStatus(status).rotulo).toBe(rotulo)
  })
})

describe('statusFinal', () => {
  it('identifica status finais corretamente', () => {
    expect(statusFinal('confirmed')).toBe(true)
    expect(statusFinal('payment_failed')).toBe(true)
    expect(statusFinal('fulfillment_failed')).toBe(true)
    expect(statusFinal('pending')).toBe(false)
    expect(statusFinal('payment_confirmed')).toBe(false)
  })
})
