import type { Order } from '@/types'
import { Badge } from './Badge'
import { infoStatus } from '@/utils/status'
import { formatarData } from '@/utils/format'

interface Props {
  pedido: Order
  selecionado: boolean
  onClick: () => void
}

export function CartaoStatus({ pedido, selecionado, onClick }: Props) {
  const info = infoStatus(pedido.status)

  return (
    <button
      onClick={onClick}
      style={{
        width: '100%',
        textAlign: 'left',
        background: selecionado ? 'var(--surface2)' : 'var(--surface)',
        border: `1px solid ${selecionado ? 'var(--primary)' : 'var(--border)'}`,
        borderRadius: 'var(--radius)',
        padding: '14px 16px',
        cursor: 'pointer',
        transition: 'border-color 0.15s, background 0.15s',
        marginBottom: '8px',
      }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '6px' }}>
        <span style={{ fontFamily: 'monospace', fontSize: '12px', color: 'var(--text-muted)' }}>
          {pedido.orderId.slice(0, 8)}…
        </span>
        <Badge cor={info.cor as 'success' | 'error' | 'warning' | 'info' | 'neutral'}>{info.rotulo}</Badge>
      </div>
      <div style={{ fontSize: '12px', color: 'var(--text-muted)' }}>
        {pedido.items.length} {pedido.items.length === 1 ? 'item' : 'itens'} · {formatarData(pedido.createdAt)}
      </div>
      {pedido.simulateFailure && (
        <div style={{ marginTop: '4px', fontSize: '11px', color: '#f87171' }}>⚡ Falha simulada</div>
      )}
    </button>
  )
}
