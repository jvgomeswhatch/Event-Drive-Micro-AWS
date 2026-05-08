import { useEffect, useState } from 'react'
import { useAuthStore } from '@/store/auth'
import { useOrdersStore } from '@/store/orders'
import { usePedidos } from '@/hooks/usePedidos'
import { CartaoStatus } from '@/components/CartaoStatus'
import { DetalhePedido } from './DetalhePedido'
import { FormularioPedido } from './FormularioPedido'
import { Botao } from '@/components/Botao'
import { Spinner } from '@/components/Spinner'

export function Dashboard() {
  const { customerId, limparAuth: sair } = useAuthStore()
  const { pedidos, pedidoSelecionado, selecionarPedido, statusLista } = useOrdersStore()
  const { carregar } = usePedidos()
  const [mostrarFormulario, setMostrarFormulario] = useState(false)

  useEffect(() => {
    carregar()
  }, [carregar])

  const pedidosOrdenados = [...pedidos].sort(
    (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime(),
  )

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100vh' }}>
      {/* Header */}
      <header
        style={{
          padding: '12px 24px',
          borderBottom: '1px solid var(--border)',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          background: 'var(--surface)',
          flexShrink: 0,
        }}
      >
        <div>
          <span style={{ fontWeight: 700, fontSize: '15px' }}>Plataforma de Pedidos</span>
          <span style={{ color: 'var(--text-muted)', fontSize: '13px', marginLeft: '12px' }}>
            {customerId}
          </span>
        </div>
        <div style={{ display: 'flex', gap: '8px' }}>
          <Botao onClick={() => setMostrarFormulario(true)}>+ Novo pedido</Botao>
          <Botao variante="ghost" onClick={sair}>Sair</Botao>
        </div>
      </header>

      {/* Layout principal */}
      <div style={{ display: 'flex', flex: 1, overflow: 'hidden' }}>
        {/* Lista de pedidos */}
        <aside
          style={{
            width: '320px',
            flexShrink: 0,
            borderRight: '1px solid var(--border)',
            overflowY: 'auto',
            padding: '16px',
            background: 'var(--surface)',
          }}
        >
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
            <h2 style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.08em' }}>
              Pedidos
            </h2>
            {statusLista === 'loading' && <Spinner tamanho={14} />}
          </div>

          {pedidosOrdenados.length === 0 && statusLista !== 'loading' && (
            <div style={{ color: 'var(--text-muted)', fontSize: '13px', textAlign: 'center', marginTop: '32px' }}>
              <div style={{ fontSize: '24px', marginBottom: '8px' }}>📭</div>
              Nenhum pedido encontrado.
              <br />
              <button
                onClick={() => setMostrarFormulario(true)}
                style={{ color: 'var(--primary)', background: 'none', border: 'none', cursor: 'pointer', marginTop: '8px', fontSize: '13px' }}
              >
                Criar o primeiro
              </button>
            </div>
          )}

          {pedidosOrdenados.map((pedido) => (
            <CartaoStatus
              key={pedido.orderId}
              pedido={pedido}
              selecionado={pedidoSelecionado?.orderId === pedido.orderId}
              onClick={() => selecionarPedido(pedido)}
            />
          ))}
        </aside>

        {/* Detalhe do pedido */}
        <main style={{ flex: 1, overflowY: 'auto', background: 'var(--bg)' }}>
          {pedidoSelecionado ? (
            <DetalhePedido pedido={pedidoSelecionado} />
          ) : (
            <div
              style={{
                height: '100%',
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                justifyContent: 'center',
                color: 'var(--text-muted)',
                fontSize: '14px',
              }}
            >
              <div style={{ fontSize: '32px', marginBottom: '12px' }}>📋</div>
              Selecione um pedido para ver os detalhes
            </div>
          )}
        </main>
      </div>

      {mostrarFormulario && (
        <FormularioPedido aoFechar={() => setMostrarFormulario(false)} />
      )}
    </div>
  )
}
