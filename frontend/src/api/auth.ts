import { apiClient } from './client'

interface TokenResponse {
  token: string
  expiresIn: number
}

export async function emitirToken(customerId: string, name: string): Promise<TokenResponse> {
  return apiClient.post<TokenResponse>('/auth/token', { customerId, name })
}
