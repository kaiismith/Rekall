import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { useBootstrap } from '@/hooks/useBootstrap'
import { useAuthStore } from '@/store/authStore'
import * as authServiceModule from '@/services/authService'

vi.mock('@/services/authService', () => ({
  authService: {
    verifyEmail: vi.fn(),
    register: vi.fn(),
    login: vi.fn(),
    logout: vi.fn(),
    refresh: vi.fn(),
    me: vi.fn(),
    resendVerification: vi.fn(),
    forgotPassword: vi.fn(),
    resetPassword: vi.fn(),
  },
}))

const mockUser = {
  id: 'user-1',
  email: 'alice@example.com',
  full_name: 'Alice',
  role: 'member',
  email_verified: true,
  created_at: '2024-01-01T00:00:00Z',
}

describe('useBootstrap', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useAuthStore.setState({ user: null, accessToken: null, isInitialised: false })
  })

  // ── No in-memory token — refresh path ─────────────────────────────────────────

  it('calls refresh() when there is no in-memory token', async () => {
    vi.mocked(authServiceModule.authService.refresh).mockResolvedValue({ access_token: 'new-tok' })
    vi.mocked(authServiceModule.authService.me).mockResolvedValue(mockUser)

    renderHook(() => useBootstrap())

    await waitFor(() => {
      expect(authServiceModule.authService.refresh).toHaveBeenCalled()
    })
  })

  it('stores user and token after a successful refresh + me()', async () => {
    vi.mocked(authServiceModule.authService.refresh).mockResolvedValue({ access_token: 'new-tok' })
    vi.mocked(authServiceModule.authService.me).mockResolvedValue(mockUser)

    renderHook(() => useBootstrap())

    await waitFor(() => {
      expect(useAuthStore.getState().user).toEqual(mockUser)
      expect(useAuthStore.getState().accessToken).toBe('new-tok')
    })
  })

  it('marks the store as initialised even when refresh() fails', async () => {
    vi.mocked(authServiceModule.authService.refresh).mockImplementation(
      () => new Promise((_, reject) => { reject(new Error('No session')) }),
    )

    renderHook(() => useBootstrap())

    await waitFor(() => {
      expect(useAuthStore.getState().isInitialised).toBe(true)
    })
  })

  it('does not set user when refresh() fails', async () => {
    vi.mocked(authServiceModule.authService.refresh).mockImplementation(
      () => new Promise((_, reject) => { reject(new Error('No session')) }),
    )

    renderHook(() => useBootstrap())

    await waitFor(() => {
      expect(useAuthStore.getState().isInitialised).toBe(true)
    })

    expect(useAuthStore.getState().user).toBeNull()
    expect(useAuthStore.getState().accessToken).toBeNull()
  })

  // ── With in-memory token — me() path ──────────────────────────────────────────

  it('calls me() instead of refresh() when an in-memory token exists', async () => {
    useAuthStore.setState({ accessToken: 'existing-tok', user: null, isInitialised: false })
    vi.mocked(authServiceModule.authService.me).mockResolvedValue(mockUser)

    renderHook(() => useBootstrap())

    await waitFor(() => {
      expect(authServiceModule.authService.me).toHaveBeenCalled()
      expect(authServiceModule.authService.refresh).not.toHaveBeenCalled()
    })
  })

  it('stores the user after a successful me() call', async () => {
    useAuthStore.setState({ accessToken: 'existing-tok', user: null, isInitialised: false })
    vi.mocked(authServiceModule.authService.me).mockResolvedValue(mockUser)

    renderHook(() => useBootstrap())

    await waitFor(() => {
      expect(useAuthStore.getState().user).toEqual(mockUser)
    })
  })

  it('marks the store as initialised even when me() fails', async () => {
    useAuthStore.setState({ accessToken: 'stale-tok', user: null, isInitialised: false })
    vi.mocked(authServiceModule.authService.me).mockImplementation(
      () => new Promise((_, reject) => { reject(new Error('401')) }),
    )

    renderHook(() => useBootstrap())

    await waitFor(() => {
      expect(useAuthStore.getState().isInitialised).toBe(true)
    })
  })

  it('always marks isInitialised true — refresh success path', async () => {
    vi.mocked(authServiceModule.authService.refresh).mockResolvedValue({ access_token: 'tok' })
    vi.mocked(authServiceModule.authService.me).mockResolvedValue(mockUser)

    renderHook(() => useBootstrap())

    await waitFor(() => {
      expect(useAuthStore.getState().isInitialised).toBe(true)
    })
  })
})
