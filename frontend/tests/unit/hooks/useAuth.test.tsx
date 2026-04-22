import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import type { ReactNode } from 'react'
import { useAuth } from '@/hooks/useAuth'
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

// useAuth calls useNavigate, which requires a Router context.
function wrapper({ children }: { children: ReactNode }) {
  return <MemoryRouter>{children}</MemoryRouter>
}

const mockUser = {
  id: 'user-1',
  email: 'alice@example.com',
  full_name: 'Alice',
  role: 'member',
  email_verified: true,
  created_at: '2024-01-01T00:00:00Z',
}

describe('useAuth', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useAuthStore.setState({ user: null, accessToken: null, isInitialised: false })
  })

  // ── Initial state ─────────────────────────────────────────────────────────────

  it('reflects unauthenticated state when store is empty', () => {
    const { result } = renderHook(() => useAuth(), { wrapper })

    expect(result.current.user).toBeNull()
    expect(result.current.accessToken).toBeNull()
    expect(result.current.isAuthenticated).toBe(false)
  })

  it('reflects authenticated state when store has a token', () => {
    useAuthStore.setState({ user: mockUser, accessToken: 'tok', isInitialised: true })

    const { result } = renderHook(() => useAuth(), { wrapper })

    expect(result.current.user).toEqual(mockUser)
    expect(result.current.isAuthenticated).toBe(true)
  })

  // ── login ─────────────────────────────────────────────────────────────────────

  it('calls authService.login with the provided credentials', async () => {
    vi.mocked(authServiceModule.authService.login).mockResolvedValue({
      user: mockUser,
      access_token: 'new-token',
    })

    const { result } = renderHook(() => useAuth(), { wrapper })

    await act(async () => {
      await result.current.login({ email: 'alice@example.com', password: 'pass' })
    })

    expect(authServiceModule.authService.login).toHaveBeenCalledWith({
      email: 'alice@example.com',
      password: 'pass',
    })
  })

  it('stores user and token in the auth store after login', async () => {
    vi.mocked(authServiceModule.authService.login).mockResolvedValue({
      user: mockUser,
      access_token: 'new-token',
    })

    const { result } = renderHook(() => useAuth(), { wrapper })

    await act(async () => {
      await result.current.login({ email: 'alice@example.com', password: 'pass' })
    })

    expect(useAuthStore.getState().user).toEqual(mockUser)
    expect(useAuthStore.getState().accessToken).toBe('new-token')
  })

  it('propagates errors thrown by authService.login', async () => {
    vi.mocked(authServiceModule.authService.login).mockRejectedValue(new Error('Invalid credentials'))

    const { result } = renderHook(() => useAuth(), { wrapper })

    await expect(
      act(async () => { await result.current.login({ email: 'x@x.com', password: 'wrong' }) }),
    ).rejects.toThrow('Invalid credentials')
  })

  // ── register ──────────────────────────────────────────────────────────────────

  it('calls authService.register with the provided payload', async () => {
    vi.mocked(authServiceModule.authService.register).mockResolvedValue(mockUser)

    const { result } = renderHook(() => useAuth(), { wrapper })

    await act(async () => {
      await result.current.register({ email: 'alice@example.com', password: 'pass', full_name: 'Alice' })
    })

    expect(authServiceModule.authService.register).toHaveBeenCalledWith({
      email: 'alice@example.com',
      password: 'pass',
      full_name: 'Alice',
    })
  })

  it('does not update the auth store after register (email verification required)', async () => {
    vi.mocked(authServiceModule.authService.register).mockResolvedValue(mockUser)

    const { result } = renderHook(() => useAuth(), { wrapper })

    await act(async () => {
      await result.current.register({ email: 'alice@example.com', password: 'pass', full_name: 'Alice' })
    })

    expect(useAuthStore.getState().user).toBeNull()
    expect(useAuthStore.getState().accessToken).toBeNull()
  })

  // ── logout ────────────────────────────────────────────────────────────────────

  it('calls authService.logout', async () => {
    vi.mocked(authServiceModule.authService.logout).mockResolvedValue()
    useAuthStore.setState({ user: mockUser, accessToken: 'tok', isInitialised: true })

    const { result } = renderHook(() => useAuth(), { wrapper })

    await act(async () => { await result.current.logout() })

    expect(authServiceModule.authService.logout).toHaveBeenCalled()
  })

  it('clears the auth store after logout', async () => {
    vi.mocked(authServiceModule.authService.logout).mockResolvedValue()
    useAuthStore.setState({ user: mockUser, accessToken: 'tok', isInitialised: true })

    const { result } = renderHook(() => useAuth(), { wrapper })

    await act(async () => { await result.current.logout() })

    expect(useAuthStore.getState().user).toBeNull()
    expect(useAuthStore.getState().accessToken).toBeNull()
  })

  it('clears the store even when authService.logout rejects', async () => {
    vi.mocked(authServiceModule.authService.logout).mockRejectedValue(new Error('Network error'))
    useAuthStore.setState({ user: mockUser, accessToken: 'tok', isInitialised: true })

    const { result } = renderHook(() => useAuth(), { wrapper })

    await act(async () => { await result.current.logout() })

    expect(useAuthStore.getState().user).toBeNull()
    expect(useAuthStore.getState().accessToken).toBeNull()
  })
})
