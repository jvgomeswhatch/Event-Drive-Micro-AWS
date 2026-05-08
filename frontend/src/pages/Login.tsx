import { useState } from 'react'
import { useAuth } from '@/hooks/useAuth'
import { Botao } from '@/components/Botao'

export function Login() {
  const { entrar, carregando, erro } = useAuth()
  const [customerId, setCustomerId] = useState('')
  const [nome, setNome] = useState('')

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!customerId.trim() || !nome.trim()) return
    await entrar(customerId.trim(), nome.trim())
  }

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '24px',
      }}
    >
      <div
        style={{
          width: '100%',
          maxWidth: '380px',
          background: 'var(--surface)',
          border: '1px solid var(--border)',
          borderRadius: 'var(--radius)',
          padding: '32px',
        }}
      >
        <h1 style={{ fontSize: '20px', fontWeight: 700, marginBottom: '8px' }}>Plataforma de Pedidos</h1>
        <p style={{ color: 'var(--text-muted)', fontSize: '13px', marginBottom: '24px' }}>
          Identifique-se para acessar o dashboard.
        </p>

        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          <div>
            <label style={{ display: 'block', fontSize: '12px', color: 'var(--text-muted)', marginBottom: '6px' }}>
              ID do Cliente
            </label>
            <input
              value={customerId}
              onChange={(e) => setCustomerId(e.target.value)}
              placeholder="ex: cliente-001"
              style={estiloInput}
              autoFocus
              required
            />
          </div>
          <div>
            <label style={{ display: 'block', fontSize: '12px', color: 'var(--text-muted)', marginBottom: '6px' }}>
              Nome
            </label>
            <input
              value={nome}
              onChange={(e) => setNome(e.target.value)}
              placeholder="ex: João Silva"
              style={estiloInput}
              required
            />
          </div>

          {erro && (
            <div
              style={{
                padding: '10px 12px',
                background: 'rgba(239,68,68,0.1)',
                border: '1px solid rgba(239,68,68,0.3)',
                borderRadius: '6px',
                color: '#f87171',
                fontSize: '13px',
              }}
            >
              {erro}
            </div>
          )}

          <Botao type="submit" carregando={carregando} style={{ width: '100%', justifyContent: 'center' }}>
            Entrar
          </Botao>
        </form>
      </div>
    </div>
  )
}

const estiloInput: React.CSSProperties = {
  width: '100%',
  padding: '9px 12px',
  background: 'var(--bg)',
  border: '1px solid var(--border)',
  borderRadius: '6px',
  color: 'var(--text)',
  fontSize: '14px',
  outline: 'none',
}
