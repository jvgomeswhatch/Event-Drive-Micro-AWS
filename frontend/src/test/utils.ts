import type { Order, TimelineEvent } from '@/types'

export function pedidoFactory(overrides: Partial<Order> = {}): Order {
  return {
    orderId: 'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    customerId: 'cliente-001',
    items: [{ productId: 'produto-001', quantity: 2 }],
    status: 'pending',
    correlationId: 'corr-123',
    createdAt: '2024-01-01T10:00:00Z',
    updatedAt: '2024-01-01T10:00:00Z',
    ...overrides,
  }
}

export function timelineFactory(overrides: Partial<TimelineEvent> = {}): TimelineEvent {
  return {
    orderId: 'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    eventId: 'payment#abc',
    service: 'payment-service',
    eventType: 'payment.succeeded',
    payload: { paymentId: 'pay-001', totalAmount: 20.0 },
    correlationId: 'corr-123',
    timestamp: '2024-01-01T10:01:00Z',
    ...overrides,
  }
}
