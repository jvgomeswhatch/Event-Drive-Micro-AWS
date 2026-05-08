import { useEffect } from 'react'
import type { Order } from '@/types'
import { useOrdersStore } from '@/store/orders'
import { usePedidos } from '@/hooks/usePedidos'
import { usePolling } from '@/hooks/usePolling'
import { Badge } from '@/components/Badge'
import { Spinner } from '@/components/Spinner'
import { TimelineEvento } from '@/components/TimelineEvento'
import { PainelCorrelation } from '@/components/PainelCorrelation'
import { infoStatus } from '@/utils/status'
import { formatarData } from '@/utils/format'

interface Props {
  pedido: Order
}

export function DetalhePedido({ pedido }: Props) {
  const { timeline, statusDetalhe } = useOrdersStore()
  const { atualizar, carregarTimeline } = usePedidos()
  const info = infoStatus(pedido.status)

  useEffect(() => {
    carregarTimeline(pedido.orderId)
  }, [pedido.orderId, pedido.status, carregarTimeline])

  usePolling({
    orderId: pedido.orderId,
    status: pedido.status,
    onAtualizar: atualizar,
  })

  return (
    <div style={{ padding: '24px', height: '100%', overflowY: 'auto' }}>
      <div style={{ marginBottom: '20px' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '8px' }}>
          <div>
            <div style={{ fontSize: '11px', color: 'var(--text-muted)', marginBottom: '4px', fontFamily: 'monospace' }}>
              Pedido #{pedido.orderId}
            </div>
            <Badge cor={info.cor as 'success' | 'error' | 'warning' | 'info' | 'neutral'}>
              {info.rotulo}
            </Badge>
          </div>
          {!['payment_failed', 'confirmed', 'fulfillment_failed'].includes(pedido.status) && (
            <div style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '12px', color: 'var(--text-muted)' }}>
              <Spinner tamanho={14} />
              Processando…
            </div>
          )}
        </div>
        <p style={{ fontSize: '13px', color: 'var(--text-muted)' }}>{info.descricao}</p>
      </div>

      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '1fr 1fr',
          gap: '8px',
          marginBottom: '20px',
        }}
      >
        <InfoItem rotulo="Cliente" valor={pedido.customerId} />
        <InfoItem rotulo="Criado em" valor={formatarData(pedido.createdAt)} />
        <InfoItem rotulo="Atualizado" valor={formatarData(pedido.updatedAt)} />
        <InfoItem rotulo="Itens" valor={`${pedido.items.length} produto(s)`} />
      </div>

      <div style={{ marginBottom: '20px' }}>
        <h3 style={{ fontSize: '12px', color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.08em', marginBottom: '10px' }}>
          Itens do pedido
        </h3>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
          {pedido.items.map((item, i) => (
            <div
              key={i}
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                padding: '8px 12px',
                background: 'var(--bg)',
                borderRadius: '6px',
                fontSize: '13px',
              }}
            >
              <span style={{ fontFamily: 'monospace' }}>{item.productId}</span>
              <span style={{ color: 'var(--text-muted)' }}>× {item.quantity}</span>
            </div>
          ))}
        </div>
      </div>

      <div>
        <h3 style={{ fontSize: '12px', color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.08em', marginBottom: '12px' }}>
          Timeline de eventos
        </h3>
        {statusDetalhe === 'loading' && timeline.length === 0 ? (
          <div style={{ display: 'flex', justifyContent: 'center', padding: '24px' }}>
            <Spinner />
          </div>
        ) : timeline.length === 0 ? (
          <div style={{ color: 'var(--text-muted)', fontSize: '13px', padding: '12px 0' }}>
            Nenhum evento registrado ainda.
          </div>
        ) : (
          <div>
            {timeline
              .slice()
              .sort((a, b) => a.timestamp.localeCompare(b.timestamp))
              .map((evento, i, arr) => (
                <TimelineEvento key={evento.eventId} evento={evento} ultimo={i === arr.length - 1} />
              ))}
          </div>
        )}
      </div>

      <PainelCorrelation pedido={pedido} />
    </div>
  )
}

function InfoItem({ rotulo, valor }: { rotulo: string; valor: string }) {
  return (
    <div style={{ padding: '10px 12px', background: 'var(--bg)', borderRadius: '6px' }}>
      <div style={{ fontSize: '11px', color: 'var(--text-muted)', marginBottom: '2px' }}>{rotulo}</div>
      <div style={{ fontSize: '13px', fontWeight: 500 }}>{valor}</div>
    </div>
  )
}
