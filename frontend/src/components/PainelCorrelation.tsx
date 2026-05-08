import type { Order } from '@/types'

interface Props {
  pedido: Order
}

export function PainelCorrelation({ pedido }: Props) {
  const debugAtivo = import.meta.env.VITE_CORRELATION_DEBUG === 'true'
  if (!debugAtivo) return null

  return (
    <div
      style={{
        background: 'rgba(99,102,241,0.08)',
        border: '1px solid rgba(99,102,241,0.2)',
        borderRadius: 'var(--radius)',
        padding: '10px 14px',
        marginTop: '12px',
        fontSize: '11px',
        fontFamily: 'monospace',
        color: 'var(--text-muted)',
      }}
    >
      <div style={{ fontWeight: 700, marginBottom: '4px', color: '#a5b4fc' }}>🔍 Correlation Debug</div>
      <div>orderId: <span style={{ color: 'var(--text)' }}>{pedido.orderId}</span></div>
      <div>correlationId: <span style={{ color: 'var(--text)' }}>{pedido.correlationId}</span></div>
    </div>
  )
}
