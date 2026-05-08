import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { CartaoStatus } from '@/components/CartaoStatus'
import { pedidoFactory } from './utils'

describe('CartaoStatus', () => {
  it('exibe os primeiros caracteres do orderId', () => {
    const pedido = pedidoFactory()
    render(<CartaoStatus pedido={pedido} selecionado={false} onClick={vi.fn()} />)
    expect(screen.getByText(/f47ac10b/)).toBeInTheDocument()
  })

  it('exibe badge de status correto', () => {
    const pedido = pedidoFactory({ status: 'confirmed' })
    render(<CartaoStatus pedido={pedido} selecionado={false} onClick={vi.fn()} />)
    expect(screen.getByText('Confirmado')).toBeInTheDocument()
  })

  it('exibe aviso de falha simulada quando ativo', () => {
    const pedido = pedidoFactory({ simulateFailure: true })
    render(<CartaoStatus pedido={pedido} selecionado={false} onClick={vi.fn()} />)
    expect(screen.getByText(/Falha simulada/)).toBeInTheDocument()
  })

  it('chama onClick ao clicar', () => {
    const onClick = vi.fn()
    const pedido = pedidoFactory()
    render(<CartaoStatus pedido={pedido} selecionado={false} onClick={onClick} />)
    fireEvent.click(screen.getByRole('button'))
    expect(onClick).toHaveBeenCalledOnce()
  })

  it('aplica borda primária quando selecionado', () => {
    const pedido = pedidoFactory()
    const { container } = render(<CartaoStatus pedido={pedido} selecionado={true} onClick={vi.fn()} />)
    const btn = container.querySelector('button')
    expect(btn?.style.borderColor).toContain('primary')
  })
})
