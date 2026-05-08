const BASE_URL = (import.meta.env.VITE_API_BASE_URL as string | undefined) || 'http://localhost:3001'
const REQUEST_TIMEOUT_MS = 10_000
const MAX_RETRIES = 2

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
    public correlationId?: string,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

interface RequestOptions extends RequestInit {
  token?: string
  correlationId?: string
  idempotencyKey?: string
  /** Override default retry count (0 = no retries) */
  retries?: number
}

async function fetchWithTimeout(url: string, options: RequestInit): Promise<Response> {
  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS)
  try {
    return await fetch(url, { ...options, signal: controller.signal })
  } finally {
    clearTimeout(timer)
  }
}

function isRetryable(status: number): boolean {
  return status === 429 || status >= 500
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { token, correlationId, idempotencyKey, retries = MAX_RETRIES, ...fetchOptions } = options

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(fetchOptions.headers as Record<string, string> | undefined),
  }

  if (token) headers['Authorization'] = `Bearer ${token}`
  if (correlationId) headers['X-Correlation-ID'] = correlationId
  if (idempotencyKey) headers['X-Idempotency-Key'] = idempotencyKey

  let lastError: unknown
  const attempts = 1 + retries

  for (let attempt = 0; attempt < attempts; attempt++) {
    if (attempt > 0) {
      await new Promise((r) => setTimeout(r, 200 * 2 ** (attempt - 1)))
    }
    try {
      const response = await fetchWithTimeout(`${BASE_URL}${path}`, { ...fetchOptions, headers })
      const responseCorrelationId = response.headers.get('X-Correlation-ID') ?? undefined

      if (!response.ok) {
        let message = `HTTP ${response.status}`
        try {
          const body = await response.json()
          message = body.error ?? message
        } catch {
          // ignore parse errors
        }
        const err = new ApiError(response.status, message, responseCorrelationId)
        if (!isRetryable(response.status) || attempt === attempts - 1) throw err
        lastError = err
        continue
      }

      return response.json() as Promise<T>
    } catch (err) {
      if (err instanceof ApiError) throw err
      lastError = err
    }
  }

  throw lastError
}

export const apiClient = {
  get: <T>(path: string, options?: RequestOptions) =>
    request<T>(path, { method: 'GET', ...options }),

  post: <T>(path: string, body: unknown, options?: RequestOptions) =>
    request<T>(path, { method: 'POST', body: JSON.stringify(body), ...options }),
}
