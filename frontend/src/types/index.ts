export type OrderStatus =
  | 'pending'
  | 'payment_confirmed'
  | 'payment_failed'
  | 'confirmed'
  | 'fulfillment_failed'

export interface OrderItem {
  productId: string
  quantity: number
  unitPrice?: number
}

export interface Order {
  orderId: string
  customerId: string
  items: OrderItem[]
  status: OrderStatus
  correlationId: string
  createdAt: string
  updatedAt: string
  simulateFailure?: boolean
}

export interface TimelineEvent {
  orderId: string
  eventId: string
  service: string
  eventType: string
  payload: Record<string, unknown>
  correlationId: string
  timestamp: string
}

export interface CreateOrderPayload {
  customerId: string
  items: OrderItem[]
  simulateFailure?: boolean
}

export interface AuthState {
  token: string | null
  customerId: string | null
}

export type AsyncStatus = 'idle' | 'loading' | 'success' | 'error'
