import type { TimelineEvent } from '@/types'
import { formatarData, formatarServico } from '@/utils/format'

interface Props {
  evento: TimelineEvent
  ultimo: boolean
}

const icones: Record<string, string> = {
  'order-service': '📋',
  'payment-service': '💳',
  'inventory-service': '📦',
  'notification-service': '🔔',
}

export function TimelineEvento({ evento, ultimo }: Props) {
  const icone = icones[evento.service] ?? '⚙️'
  const falhou = evento.eventType.includes('failed')

  return (
    <div style={{ display: 'flex', gap: '12px' }}>
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
        <div
          style={{
            width: '32px',
            height: '32px',
            borderRadius: '50%',
            background: falhou ? 'rgba(239,68,68,0.15)' : 'rgba(99,102,241,0.15)',
            border: `2px solid ${falhou ? '#ef4444' : '#6366f1'}`,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: '14px',
            flexShrink: 0,
          }}
        >
          {icone}
        </div>
        {!ultimo && (
          <div style={{ width: '2px', flex: 1, background: 'var(--border)', margin: '4px 0' }} />
        )}
      </div>
      <div style={{ paddingBottom: ultimo ? 0 : '16px', flex: 1 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
          <div>
            <div style={{ fontWeight: 600, fontSize: '13px' }}>{formatarServico(evento.service)}</div>
            <div style={{ fontSize: '12px', color: 'var(--text-muted)', fontFamily: 'monospace' }}>
              {evento.eventType}
            </div>
          </div>
          <div style={{ fontSize: '11px', color: 'var(--text-muted)', whiteSpace: 'nowrap' }}>
            {formatarData(evento.timestamp)}
          </div>
        </div>
        {evento.payload && Object.keys(evento.payload).length > 0 && (
          <details style={{ marginTop: '6px' }}>
            <summary style={{ fontSize: '11px', color: 'var(--text-muted)', cursor: 'pointer' }}>
              payload
            </summary>
            <pre
              style={{
                marginTop: '4px',
                padding: '8px',
                background: 'var(--bg)',
                borderRadius: '4px',
                fontSize: '11px',
                color: 'var(--text-muted)',
                overflow: 'auto',
                maxHeight: '120px',
              }}
            >
              {JSON.stringify(evento.payload, null, 2)}
            </pre>
          </details>
        )}
      </div>
    </div>
  )
}
