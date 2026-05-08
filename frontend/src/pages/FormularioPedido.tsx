import { useState } from 'react'
import { useOrdersStore } from '@/store/orders'
import { usePedidos } from '@/hooks/usePedidos'
import { useAuthStore } from '@/store/auth'
import { Botao } from '@/components/Botao'
import type { OrderItem } from '@/types'

interface Props {
  aoFechar: () => void
}

export function FormularioPedido({ aoFechar }: Props) {
  const { customerId } = useAuthStore()
  const { criar } = usePedidos()
  const { statusCriacao, erroCriacao } = useOrdersStore()

  const [itens, setItens] = useState<OrderItem[]>([{ productId: 'prod-001', quantity: 1, unitPrice: 299.99 }])
  const [simularFalha, setSimularFalha] = useState(false)

  function adicionarItem() {
    setItens((prev) => [...prev, { productId: '', quantity: 1 }])
  }

  function removerItem(idx: number) {
    setItens((prev) => prev.filter((_, i) => i !== idx))
  }

  function atualizarItem(idx: number, campo: keyof OrderItem, valor: string | number) {
    setItens((prev) => prev.map((item, i) => (i === idx ? { ...item, [campo]: valor } : item)))
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!customerId) return
    const orderId = await criar({
      customerId,
      items: itens.filter((i) => i.productId.trim()),
      simulateFailure: simularFalha,
    })
    if (orderId) aoFechar()
  }

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0,0,0,0.6)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 100,
        padding: '24px',
      }}
      onClick={(e) => e.target === e.currentTarget && aoFechar()}
    >
      <div
        style={{
          width: '100%',
          maxWidth: '560px',
          background: 'var(--surface)',
          border: '1px solid var(--border)',
          borderRadius: 'var(--radius)',
          padding: '24px',
        }}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '20px' }}>
          <h2 style={{ fontSize: '16px', fontWeight: 700 }}>Novo Pedido</h2>
          <button onClick={aoFechar} style={{ background: 'none', border: 'none', color: 'var(--text-muted)', cursor: 'pointer', fontSize: '18px' }}>
            ✕
          </button>
        </div>

        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '8px' }}>
              <label style={{ fontSize: '12px', color: 'var(--text-muted)' }}>Itens</label>
              <Botao variante="ghost" type="button" onClick={adicionarItem} style={{ fontSize: '12px', padding: '4px 8px' }}>
                + Adicionar
              </Botao>
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
              {itens.map((item, idx) => (
                <div key={idx} style={{ display: 'grid', gridTemplateColumns: 'minmax(120px, 2fr) minmax(60px, 1fr) minmax(80px, 1fr) auto', gap: '8px', alignItems: 'center' }}>
                  <input
                    value={item.productId}
                    onChange={(e) => atualizarItem(idx, 'productId', e.target.value)}
                    placeholder="ID do produto"
                    required
                    style={estiloInput}
                  />
                  <input
                    type="number"
                    value={item.quantity}
                    min={1}
                    max={1000}
                    placeholder="Qtd"
                    onChange={(e) => atualizarItem(idx, 'quantity', parseInt(e.target.value) || 1)}
                    style={estiloInput}
                  />
                  <input
                    type="number"
                    value={item.unitPrice ?? ''}
                    min={0.01}
                    step={0.01}
                    placeholder="Preço"
                    onChange={(e) => atualizarItem(idx, 'unitPrice', parseFloat(e.target.value) || 0)}
                    style={estiloInput}
                  />
                  {itens.length > 1 && (
                    <Botao variante="ghost" type="button" onClick={() => removerItem(idx)} style={{ padding: '8px', color: '#f87171' }}>
                      ✕
                    </Botao>
                  )}
                </div>
              ))}
            </div>
          </div>

          <div
            style={{
              padding: '12px',
              background: 'rgba(239,68,68,0.06)',
              border: '1px solid rgba(239,68,68,0.2)',
              borderRadius: '6px',
            }}
          >
            <label style={{ display: 'flex', alignItems: 'center', gap: '10px', cursor: 'pointer' }}>
              <input
                type="checkbox"
                checked={simularFalha}
                onChange={(e) => setSimularFalha(e.target.checked)}
                style={{ width: '16px', height: '16px' }}
              />
              <div>
                <div style={{ fontSize: '13px', fontWeight: 600, color: '#f87171' }}>⚡ Simular falha de pagamento</div>
                <div style={{ fontSize: '11px', color: 'var(--text-muted)' }}>
                  Força falha no payment-service para demonstrar o fluxo de DLQ
                </div>
              </div>
            </label>
          </div>

          {erroCriacao && (
            <div style={{ padding: '10px 12px', background: 'rgba(239,68,68,0.1)', border: '1px solid rgba(239,68,68,0.3)', borderRadius: '6px', color: '#f87171', fontSize: '13px' }}>
              {erroCriacao}
            </div>
          )}

          <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
            <Botao variante="ghost" type="button" onClick={aoFechar}>Cancelar</Botao>
            <Botao type="submit" carregando={statusCriacao === 'loading'}>Criar pedido</Botao>
          </div>
        </form>
      </div>
    </div>
  )
}

const estiloInput: React.CSSProperties = {
  padding: '9px 12px',
  background: 'var(--bg)',
  border: '1px solid var(--border)',
  borderRadius: '6px',
  color: 'var(--text)',
  fontSize: '14px',
  outline: 'none',
}
