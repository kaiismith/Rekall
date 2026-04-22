import axios, { type AxiosError, type AxiosInstance, type AxiosResponse } from 'axios'
import { API_BASE_URL } from '@/constants'

/** Typed API error with a machine-readable code. */
export class ApiError extends Error {
  readonly code: string
  readonly status: number
  readonly details: unknown

  constructor(code: string, message: string, status: number, details?: unknown) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.status = status
    this.details = details
  }
}

/** Axios instance with default config. */
export const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  headers: { 'Content-Type': 'application/json' },
  timeout: 30_000,
})

// ─── Request interceptor ──────────────────────────────────────────────────────
apiClient.interceptors.request.use((config) => {
  // Attach a request ID for correlation with backend logs.
  config.headers['X-Request-ID'] = crypto.randomUUID()
  return config
})

// ─── Response interceptor ─────────────────────────────────────────────────────
apiClient.interceptors.response.use(
  (response: AxiosResponse) => response,
  (error: AxiosError<{ error?: { code?: string; message?: string; details?: unknown } }>) => {
    const status = error.response?.status ?? 0
    const payload = error.response?.data?.error

    if (payload) {
      throw new ApiError(
        payload.code ?? 'UNKNOWN_ERROR',
        payload.message ?? error.message,
        status,
        payload.details,
      )
    }

    if (error.code === 'ECONNABORTED') {
      throw new ApiError('TIMEOUT', 'Request timed out. Please try again.', 408)
    }

    if (!error.response) {
      throw new ApiError('NETWORK_ERROR', 'Network error. Check your connection.', 0)
    }

    throw new ApiError('UNKNOWN_ERROR', error.message, status)
  },
)
