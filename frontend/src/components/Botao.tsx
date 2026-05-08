import { Spinner } from './Spinner'

type Variante = 'primario' | 'secundario' | 'perigo' | 'ghost'

interface Props extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variante?: Variante
  carregando?: boolean
}

const estilos: Record<Variante, React.CSSProperties> = {
  primario: { background: '#6366f1', color: '#fff', border: '1px solid #6366f1' },
  secundario: { background: 'transparent', color: '#6366f1', border: '1px solid #6366f1' },
  perigo: { background: '#ef4444', color: '#fff', border: '1px solid #ef4444' },
  ghost: { background: 'transparent', color: '#94a3b8', border: '1px solid #2e3250' },
}

export function Botao({ variante = 'primario', carregando, children, disabled, style, ...props }: Props) {
  return (
    <button
      disabled={disabled || carregando}
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: '8px',
        padding: '8px 16px',
        borderRadius: '6px',
        fontSize: '14px',
        fontWeight: 500,
        cursor: disabled || carregando ? 'not-allowed' : 'pointer',
        opacity: disabled || carregando ? 0.6 : 1,
        transition: 'opacity 0.15s',
        ...estilos[variante],
        ...style,
      }}
      {...props}
    >
      {carregando && <Spinner tamanho={14} />}
      {children}
    </button>
  )
}
