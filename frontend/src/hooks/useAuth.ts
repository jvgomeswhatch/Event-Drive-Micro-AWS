import { useState } from 'react'
import { useAuthStore } from '@/store/auth'
import { emitirToken } from '@/api/auth'
import { ApiError } from '@/api/client'

export function useAuth() {
  const { token, customerId, setAuth, limparAuth } = useAuthStore()
  const [carregando, setCarregando] = useState(false)
  const [erro, setErro] = useState<string | null>(null)

  async function entrar(customerId: string, nome: string) {
    setCarregando(true)
    setErro(null)
    try {
      const res = await emitirToken(customerId, nome)
      setAuth(res.token, customerId)
    } catch (err) {
      setErro(err instanceof ApiError ? err.message : 'Falha ao autenticar')
    } finally {
      setCarregando(false)
    }
  }

  return { token, customerId, autenticado: !!token, entrar, sair: limparAuth, carregando, erro }
}
