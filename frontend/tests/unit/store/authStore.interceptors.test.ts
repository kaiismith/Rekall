import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { AxiosError, AxiosHeaders, type InternalAxiosRequestConfig } from 'axios'
import { apiClient, ApiError } from '@/services/api'
import { useAuthStore } from '@/store/authStore'

// Mock authService — used by the 401 refresh interceptor.
vi.mock('@/services/authService', () => ({
  authService: {
    refresh: vi.fn(),
  },
}))

import { authService } from '@/services/authService'

// ── Helpers ──────────────────────────────────────────────────────────────────

const originalAdapter = apiClient.defaults.adapter

afterEach(() => {
  apiClient.defaults.adapter = originalAdapter
  useAuthStore.setState({ user: null, accessToken: null, isInitialised: false })
  vi.clearAllMocks()
})

/**
 * Extract the interceptor handler functions from the apiClient.
 * api.ts interceptors are at index 0, authStore's at index 1.
 */
function getAuthInterceptors() {
  const reqHandlers: Array<{ fulfilled: (c: InternalAxiosRequestConfig) => InternalAxiosRequestConfig }> = []
  ;(apiClient.interceptors.request as any).forEach((h: any) => reqHandlers.push(h))

  const resHandlers: Array<{ fulfilled: (r: any) => any; rejected: (e: unknown) => any }> = []
  ;(apiClient.interceptors.response as any).forEach((h: any) => resHandlers.push(h))

  // authStore's interceptors are added after api.ts's, so they're the last ones.
  return {
    requestHandler: reqHandlers[reqHandlers.length - 1]?.fulfilled,
    // Find the handler that does 401 refresh — it's the last response handler
    responseErrorHandler: resHandlers[resHandlers.length - 1]?.rejected,
  }
}

function makeConfig(url = '/test', extras: Record<string, unknown> = {}): InternalAxiosRequestConfig {
  const headers = new AxiosHeaders()
  return { url, headers, ...extras } as InternalAxiosRequestConfig
}

function make401Error(url = '/test', retried = false): AxiosError {
  const config = makeConfig(url) as any
  if (retried) config._retried = true
  return new AxiosError(
    'Unauthorized',
    'ERR_BAD_REQUEST',
    config,
    {},
    { status: 401, data: {}, headers: {}, statusText: 'Unauthorized', config },
  )
}

// ── Request interceptor ──────────────────────────────────────────────────────

describe('authStore request interceptor', () => {
  it('attaches Authorization header when accessToken is set', () => {
    useAuthStore.setState({ accessToken: 'tok-abc' })
    const { requestHandler } = getAuthInterceptors()
    const config = makeConfig()

    const result = requestHandler(config)

    expect(result.headers.get('Authorization')).toBe('Bearer tok-abc')
  })

  it('does not attach Authorization header when no token', () => {
    useAuthStore.setState({ accessToken: null })
    const { requestHandler } = getAuthInterceptors()
    const config = makeConfig()

    const result = requestHandler(config)

    expect(result.headers.get('Authorization')).toBeFalsy()
  })

  it('reflects latest token from the store', () => {
    const { requestHandler } = getAuthInterceptors()

    useAuthStore.setState({ accessToken: 'first' })
    const r1 = requestHandler(makeConfig())
    expect(r1.headers.get('Authorization')).toBe('Bearer first')

    useAuthStore.setState({ accessToken: 'second' })
    const r2 = requestHandler(makeConfig())
    expect(r2.headers.get('Authorization')).toBe('Bearer second')
  })
})

// ── Response interceptor ─────────────────────────────────────────────────────

describe('authStore response interceptor (401 refresh)', () => {
  beforeEach(() => vi.clearAllMocks())

  it('passes through non-AxiosError', async () => {
    const { responseErrorHandler } = getAuthInterceptors()
    const plainError = new Error('Something else')

    await expect(responseErrorHandler(plainError)).rejects.toBe(plainError)
  })

  it('passes through AxiosError without config', async () => {
    const { responseErrorHandler } = getAuthInterceptors()
    const error = new AxiosError('no config')
    ;(error as any).config = undefined

    await expect(responseErrorHandler(error)).rejects.toBe(error)
  })

  it('passes through non-401 AxiosError', async () => {
    const { responseErrorHandler } = getAuthInterceptors()
    const error = new AxiosError(
      'Forbidden',
      'ERR_BAD_REQUEST',
      makeConfig() as any,
      {},
      { status: 403, data: {}, headers: {}, statusText: 'Forbidden', config: makeConfig() as any },
    )

    await expect(responseErrorHandler(error)).rejects.toBe(error)
  })

  it('passes through 401 on skipped paths (/auth/login)', async () => {
    const { responseErrorHandler } = getAuthInterceptors()
    const error = make401Error('/auth/login')

    await expect(responseErrorHandler(error)).rejects.toBe(error)
  })

  it('passes through 401 on skipped paths (/auth/register)', async () => {
    const { responseErrorHandler } = getAuthInterceptors()
    const error = make401Error('/auth/register')

    await expect(responseErrorHandler(error)).rejects.toBe(error)
  })

  it('passes through 401 on skipped paths (/auth/refresh)', async () => {
    const { responseErrorHandler } = getAuthInterceptors()
    const error = make401Error('/auth/refresh')

    await expect(responseErrorHandler(error)).rejects.toBe(error)
  })

  it('passes through already-retried 401', async () => {
    const { responseErrorHandler } = getAuthInterceptors()
    const error = make401Error('/api/data', true)

    await expect(responseErrorHandler(error)).rejects.toBe(error)
  })

  it('attempts refresh on 401 and retries original request on success', async () => {
    const mockUser = { id: 'u1', email: 'a@b.com', full_name: 'A', role: 'member', email_verified: true, created_at: '' }
    useAuthStore.setState({ user: mockUser, accessToken: 'old-tok' })

    vi.mocked(authService.refresh).mockResolvedValue({ access_token: 'new-tok' })

    // Mock apiClient to capture the retried request
    const retryResult = { data: 'retried', status: 200 }
    apiClient.defaults.adapter = (config: any) => {
      if (config._retried) {
        return Promise.resolve({ ...retryResult, statusText: 'OK', headers: {}, config })
      }
      return Promise.reject(make401Error(config.url))
    }

    const { responseErrorHandler } = getAuthInterceptors()
    const error = make401Error('/api/data')

    // The handler retries via apiClient(originalRequest) which goes through the adapter
    const result = await responseErrorHandler(error)

    expect(authService.refresh).toHaveBeenCalledTimes(1)
    expect(useAuthStore.getState().accessToken).toBe('new-tok')
    expect(result.data).toBe('retried')
  })

  it('clears auth and throws ApiError when refresh fails', async () => {
    const mockUser = { id: 'u1', email: 'a@b.com', full_name: 'A', role: 'member', email_verified: true, created_at: '' }
    useAuthStore.setState({ user: mockUser, accessToken: 'old-tok' })

    vi.mocked(authService.refresh).mockRejectedValue(new Error('Refresh failed'))

    const { responseErrorHandler } = getAuthInterceptors()
    const error = make401Error('/api/data')

    await expect(responseErrorHandler(error)).rejects.toThrow(ApiError)

    try {
      await responseErrorHandler(make401Error('/api/other'))
    } catch (e) {
      // The first call already cleared auth and threw
    }

    // Auth should be cleared
    expect(useAuthStore.getState().user).toBeNull()
    expect(useAuthStore.getState().accessToken).toBeNull()
  })

  it('does not call setAuth if user is null during refresh', async () => {
    useAuthStore.setState({ user: null, accessToken: 'old-tok' })
    vi.mocked(authService.refresh).mockResolvedValue({ access_token: 'new-tok' })

    apiClient.defaults.adapter = (config: any) => {
      if (config._retried) {
        return Promise.resolve({ data: 'ok', status: 200, statusText: 'OK', headers: {}, config })
      }
      return Promise.reject(make401Error(config.url))
    }

    const { responseErrorHandler } = getAuthInterceptors()
    await responseErrorHandler(make401Error('/api/data'))

    // setAuth was NOT called because user was null → token remains the old value
    expect(useAuthStore.getState().accessToken).toBe('old-tok')
  })

  it('queues concurrent 401 requests and replays them after refresh', async () => {
    const mockUser = { id: 'u1', email: 'a@b.com', full_name: 'A', role: 'member', email_verified: true, created_at: '' }
    useAuthStore.setState({ user: mockUser, accessToken: 'old-tok' })

    let resolveRefresh!: (value: { access_token: string }) => void
    vi.mocked(authService.refresh).mockReturnValue(
      new Promise((r) => { resolveRefresh = r }),
    )

    apiClient.defaults.adapter = (config: any) => {
      if (config._retried || config.headers?.get?.('Authorization')?.includes('new-tok')) {
        return Promise.resolve({ data: `ok-${config.url}`, status: 200, statusText: 'OK', headers: {}, config })
      }
      return Promise.reject(make401Error(config.url))
    }

    const { responseErrorHandler } = getAuthInterceptors()

    // Fire first 401 — starts the refresh
    const p1 = responseErrorHandler(make401Error('/api/a'))
    // Fire second 401 — should be queued
    const p2 = responseErrorHandler(make401Error('/api/b'))

    // Only one refresh should be in flight
    expect(authService.refresh).toHaveBeenCalledTimes(1)

    // Resolve the refresh
    resolveRefresh({ access_token: 'new-tok' })

    const [r1, r2] = await Promise.all([p1, p2])
    expect(r1.data).toBe('ok-/api/a')
    expect(r2.data).toBe('ok-/api/b')
  })
})
