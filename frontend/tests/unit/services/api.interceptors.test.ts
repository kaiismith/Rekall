import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import axios, { AxiosError, AxiosHeaders } from 'axios'
import { apiClient, ApiError } from '@/services/api'

// ── Helpers ──────────────────────────────────────────────────────────────────

// Store the original adapter so we can restore it after each test.
const originalAdapter = apiClient.defaults.adapter

afterEach(() => {
  apiClient.defaults.adapter = originalAdapter
})

/** Build a minimal AxiosError for injecting into the interceptor chain. */
function makeAxiosError(opts: {
  status?: number
  data?: unknown
  code?: string
  message?: string
  url?: string
}): AxiosError {
  const headers = new AxiosHeaders()
  const config = { headers, url: opts.url ?? '/test' }

  const error = new AxiosError(
    opts.message ?? 'Request failed',
    opts.code ?? 'ERR_BAD_REQUEST',
    config as any,
    {},
    opts.status
      ? { status: opts.status, data: opts.data ?? {}, headers: {}, statusText: 'Error', config: config as any }
      : undefined,
  )
  return error
}

// ── Request interceptor: X-Request-ID ────────────────────────────────────────

describe('api request interceptor', () => {
  beforeEach(() => vi.clearAllMocks())

  it('attaches X-Request-ID header to outgoing requests', async () => {
    let capturedHeaders: Record<string, string> | undefined

    apiClient.defaults.adapter = (config: any) => {
      capturedHeaders = config.headers instanceof AxiosHeaders
        ? Object.fromEntries([...config.headers])
        : config.headers
      return Promise.resolve({ data: {}, status: 200, statusText: 'OK', headers: {}, config })
    }

    await apiClient.get('/test-id')

    expect(capturedHeaders).toBeDefined()
    // X-Request-ID should be a UUID v4 string
    const id = capturedHeaders?.['x-request-id'] ?? capturedHeaders?.['X-Request-ID']
    expect(id).toBeDefined()
    expect(typeof id).toBe('string')
    expect(id!.length).toBeGreaterThan(0)
  })

  it('generates unique X-Request-ID per request', async () => {
    const ids: string[] = []

    apiClient.defaults.adapter = (config: any) => {
      const h = config.headers instanceof AxiosHeaders
        ? Object.fromEntries([...config.headers])
        : config.headers
      ids.push(h?.['x-request-id'] ?? h?.['X-Request-ID'] ?? '')
      return Promise.resolve({ data: {}, status: 200, statusText: 'OK', headers: {}, config })
    }

    await apiClient.get('/a')
    await apiClient.get('/b')

    expect(ids[0]).not.toBe(ids[1])
  })
})

// ── Response interceptor: error transforms ───────────────────────────────────

describe('api response interceptor', () => {
  beforeEach(() => vi.clearAllMocks())

  it('passes through successful responses', async () => {
    apiClient.defaults.adapter = (config: any) =>
      Promise.resolve({ data: { ok: true }, status: 200, statusText: 'OK', headers: {}, config })

    const res = await apiClient.get('/success')
    expect(res.data).toEqual({ ok: true })
  })

  it('transforms structured error payload to ApiError', async () => {
    apiClient.defaults.adapter = () =>
      Promise.reject(
        makeAxiosError({
          status: 422,
          data: { error: { code: 'VALIDATION_ERROR', message: 'Email is invalid', details: { field: 'email' } } },
        }),
      )

    await expect(apiClient.get('/fail')).rejects.toThrow(ApiError)

    try {
      await apiClient.get('/fail')
    } catch (err) {
      const e = err as ApiError
      expect(e.code).toBe('VALIDATION_ERROR')
      expect(e.message).toBe('Email is invalid')
      expect(e.status).toBe(422)
      expect(e.details).toEqual({ field: 'email' })
    }
  })

  it('uses fallback code/message when structured payload has no code or message', async () => {
    apiClient.defaults.adapter = () =>
      Promise.reject(
        makeAxiosError({
          status: 400,
          message: 'Bad Request',
          data: { error: {} },
        }),
      )

    try {
      await apiClient.get('/fail')
    } catch (err) {
      const e = err as ApiError
      expect(e.code).toBe('UNKNOWN_ERROR')
      expect(e.message).toBe('Bad Request')
      expect(e.status).toBe(400)
    }
  })

  it('transforms ECONNABORTED to TIMEOUT ApiError', async () => {
    apiClient.defaults.adapter = () =>
      Promise.reject(
        makeAxiosError({ code: 'ECONNABORTED', message: 'timeout of 30000ms exceeded' }),
      )

    try {
      await apiClient.get('/timeout')
    } catch (err) {
      const e = err as ApiError
      expect(e.code).toBe('TIMEOUT')
      expect(e.message).toBe('Request timed out. Please try again.')
      expect(e.status).toBe(408)
    }
  })

  it('transforms no-response error to NETWORK_ERROR', async () => {
    // An error with no `.response` and no ECONNABORTED (e.g. DNS failure)
    const error = new AxiosError(
      'Network Error',
      'ERR_NETWORK',
      { headers: new AxiosHeaders() } as any,
      {},
      undefined, // no response
    )

    apiClient.defaults.adapter = () => Promise.reject(error)

    try {
      await apiClient.get('/network')
    } catch (err) {
      const e = err as ApiError
      expect(e.code).toBe('NETWORK_ERROR')
      expect(e.message).toBe('Network error. Check your connection.')
      expect(e.status).toBe(0)
    }
  })

  it('transforms error with response but no payload to UNKNOWN_ERROR', async () => {
    apiClient.defaults.adapter = () =>
      Promise.reject(
        makeAxiosError({ status: 503, data: {}, message: 'Service Unavailable' }),
      )

    try {
      await apiClient.get('/503')
    } catch (err) {
      const e = err as ApiError
      expect(e.code).toBe('UNKNOWN_ERROR')
      expect(e.message).toBe('Service Unavailable')
      expect(e.status).toBe(503)
    }
  })
})
