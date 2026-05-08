import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { apiClient, ApiError } from '@/api/client'

describe('apiClient', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('retorna dados em caso de sucesso', async () => {
    const dados = { token: 'abc123' }
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify(dados), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    const resultado = await apiClient.get('/auth/token')
    expect(resultado).toEqual(dados)
  })

  it('lança ApiError com status em caso de falha', async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ error: 'Não autorizado' }), {
        status: 401,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    await expect(apiClient.get('/orders')).rejects.toThrow(ApiError)
    await expect(apiClient.get('/orders')).rejects.toMatchObject({ status: 401 })
  })

  it('inclui Authorization header quando token fornecido', async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({}), { status: 200 }),
    )
    await apiClient.get('/orders', { token: 'meu-token' })
    const chamada = vi.mocked(fetch).mock.calls[0]
    const headers = chamada[1]?.headers as Record<string, string>
    expect(headers['Authorization']).toBe('Bearer meu-token')
  })

  it('inclui X-Idempotency-Key quando fornecido', async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({}), { status: 200 }),
    )
    await apiClient.post('/orders', {}, { idempotencyKey: 'chave-123' })
    const chamada = vi.mocked(fetch).mock.calls[0]
    const headers = chamada[1]?.headers as Record<string, string>
    expect(headers['X-Idempotency-Key']).toBe('chave-123')
  })
})
