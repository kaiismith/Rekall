import { describe, it, expect, vi, beforeEach } from 'vitest'
import { authService } from '@/services/authService'

vi.mock('@/services/api', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
  },
}))

import { apiClient } from '@/services/api'

const mockUser = {
  id: 'user-1',
  email: 'alice@example.com',
  name: 'Alice',
  email_verified: true,
  created_at: '2026-01-01T00:00:00Z',
}

const mockLoginResponse = {
  access_token: 'tok-abc',
  user: mockUser,
}

describe('authService', () => {
  beforeEach(() => vi.clearAllMocks())

  // ── register ────────────────────────────────────────────────────────────────
  it('register() calls POST /auth/register and returns the user', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({ data: { data: mockUser } })

    const payload = { email: 'alice@example.com', password: 'secret', name: 'Alice' }
    const result = await authService.register(payload)

    expect(apiClient.post).toHaveBeenCalledWith('/auth/register', payload)
    expect(result).toEqual(mockUser)
  })

  // ── login ───────────────────────────────────────────────────────────────────
  it('login() calls POST /auth/login and returns LoginResponse', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({ data: { data: mockLoginResponse } })

    const payload = { email: 'alice@example.com', password: 'secret' }
    const result = await authService.login(payload)

    expect(apiClient.post).toHaveBeenCalledWith('/auth/login', payload)
    expect(result).toEqual(mockLoginResponse)
  })

  // ── logout ──────────────────────────────────────────────────────────────────
  it('logout() calls POST /auth/logout', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({})

    await authService.logout()

    expect(apiClient.post).toHaveBeenCalledWith('/auth/logout')
  })

  it('logout() resolves to undefined (void)', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({})
    const result = await authService.logout()
    expect(result).toBeUndefined()
  })

  // ── refresh ─────────────────────────────────────────────────────────────────
  it('refresh() calls POST /auth/refresh and returns access_token', async () => {
    const token = { access_token: 'new-tok' }
    vi.mocked(apiClient.post).mockResolvedValue({ data: { data: token } })

    const result = await authService.refresh()

    expect(apiClient.post).toHaveBeenCalledWith('/auth/refresh')
    expect(result).toEqual(token)
  })

  // ── me ──────────────────────────────────────────────────────────────────────
  it('me() calls GET /auth/me and returns the user', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: { data: mockUser } })

    const result = await authService.me()

    expect(apiClient.get).toHaveBeenCalledWith('/auth/me')
    expect(result).toEqual(mockUser)
  })

  // ── verifyEmail ─────────────────────────────────────────────────────────────
  it('verifyEmail() calls GET /auth/verify with token param', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({})

    await authService.verifyEmail('tok-verify')

    expect(apiClient.get).toHaveBeenCalledWith('/auth/verify', { params: { token: 'tok-verify' } })
  })

  it('verifyEmail() resolves to undefined (void)', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({})
    const result = await authService.verifyEmail('tok')
    expect(result).toBeUndefined()
  })

  // ── resendVerification ──────────────────────────────────────────────────────
  it('resendVerification() calls POST /auth/verify/resend', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({})

    const payload = { email: 'alice@example.com' }
    await authService.resendVerification(payload)

    expect(apiClient.post).toHaveBeenCalledWith('/auth/verify/resend', payload)
  })

  // ── forgotPassword ──────────────────────────────────────────────────────────
  it('forgotPassword() calls POST /auth/password/forgot', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({})

    const payload = { email: 'alice@example.com' }
    await authService.forgotPassword(payload)

    expect(apiClient.post).toHaveBeenCalledWith('/auth/password/forgot', payload)
  })

  // ── resetPassword ───────────────────────────────────────────────────────────
  it('resetPassword() calls POST /auth/password/reset', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({})

    const payload = { token: 'reset-tok', password: 'newSecret' }
    await authService.resetPassword(payload)

    expect(apiClient.post).toHaveBeenCalledWith('/auth/password/reset', payload)
  })

  it('resetPassword() resolves to undefined (void)', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({})
    const result = await authService.resetPassword({ token: 't', password: 'p' })
    expect(result).toBeUndefined()
  })
})
