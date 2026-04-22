import { describe, it, expect, beforeEach } from 'vitest'
import { useAuthStore } from '@/store/authStore'
import type { User } from '@/types/auth'

const mockUser: User = {
  id: 'user-1',
  email: 'alice@example.com',
  full_name: 'Alice',
  is_verified: true,
  created_at: '2026-01-01T00:00:00Z',
}

describe('authStore', () => {
  beforeEach(() => {
    // Reset to initial state between tests.
    useAuthStore.setState({ user: null, accessToken: null, isInitialised: false })
  })

  it('starts with null user, null token, and isInitialised=false', () => {
    const { user, accessToken, isInitialised } = useAuthStore.getState()
    expect(user).toBeNull()
    expect(accessToken).toBeNull()
    expect(isInitialised).toBe(false)
  })

  it('setAuth() stores user and token', () => {
    useAuthStore.getState().setAuth(mockUser, 'tok-abc')
    const { user, accessToken } = useAuthStore.getState()
    expect(user).toEqual(mockUser)
    expect(accessToken).toBe('tok-abc')
  })

  it('clearAuth() removes user and token', () => {
    useAuthStore.getState().setAuth(mockUser, 'tok-abc')
    useAuthStore.getState().clearAuth()
    const { user, accessToken } = useAuthStore.getState()
    expect(user).toBeNull()
    expect(accessToken).toBeNull()
  })

  it('setInitialised() sets isInitialised to true', () => {
    useAuthStore.getState().setInitialised()
    expect(useAuthStore.getState().isInitialised).toBe(true)
  })

  it('setAuth() does not affect isInitialised', () => {
    useAuthStore.getState().setAuth(mockUser, 'tok-abc')
    expect(useAuthStore.getState().isInitialised).toBe(false)
  })

  it('clearAuth() does not affect isInitialised', () => {
    useAuthStore.getState().setInitialised()
    useAuthStore.getState().setAuth(mockUser, 'tok-abc')
    useAuthStore.getState().clearAuth()
    expect(useAuthStore.getState().isInitialised).toBe(true)
  })

  it('sequential setAuth() calls overwrite with latest values', () => {
    const otherUser: User = { ...mockUser, id: 'user-2', email: 'bob@example.com' }
    useAuthStore.getState().setAuth(mockUser, 'tok-1')
    useAuthStore.getState().setAuth(otherUser, 'tok-2')
    const { user, accessToken } = useAuthStore.getState()
    expect(user).toEqual(otherUser)
    expect(accessToken).toBe('tok-2')
  })
})
