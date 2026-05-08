import { clsx } from 'clsx'

type Cor = 'success' | 'error' | 'warning' | 'info' | 'neutral'

interface Props {
  cor: Cor
  children: React.ReactNode
}

const estilos: Record<Cor, string> = {
  success: 'badge--success',
  error: 'badge--error',
  warning: 'badge--warning',
  info: 'badge--info',
  neutral: 'badge--neutral',
}

export function Badge({ cor, children }: Props) {
  return (
    <span
      className={clsx('badge', estilos[cor])}
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: '4px',
        padding: '2px 10px',
        borderRadius: '999px',
        fontSize: '12px',
        fontWeight: 600,
        letterSpacing: '0.02em',
        background: corFundo[cor],
        color: corTexto[cor],
        border: `1px solid ${corBorda[cor]}`,
      }}
    >
      {children}
    </span>
  )
}

const corFundo: Record<Cor, string> = {
  success: 'rgba(34,197,94,0.15)',
  error: 'rgba(239,68,68,0.15)',
  warning: 'rgba(245,158,11,0.15)',
  info: 'rgba(59,130,246,0.15)',
  neutral: 'rgba(148,163,184,0.15)',
}
const corTexto: Record<Cor, string> = {
  success: '#4ade80',
  error: '#f87171',
  warning: '#fbbf24',
  info: '#60a5fa',
  neutral: '#94a3b8',
}
const corBorda: Record<Cor, string> = {
  success: 'rgba(34,197,94,0.3)',
  error: 'rgba(239,68,68,0.3)',
  warning: 'rgba(245,158,11,0.3)',
  info: 'rgba(59,130,246,0.3)',
  neutral: 'rgba(148,163,184,0.3)',
}
