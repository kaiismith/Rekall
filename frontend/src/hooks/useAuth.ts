import { useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '@/store/authStore'
import { authService } from '@/services/authService'
import { ROUTES } from '@/constants'
import type { LoginPayload, RegisterPayload } from '@/types/auth'

/**
 * Convenience hook that exposes auth state and login/logout/register actions.
 * Components should use this rather than accessing the store and service directly.
 */
export function useAuth() {
  const navigate = useNavigate()
  const { user, accessToken, isInitialised, setAuth, clearAuth } = useAuthStore()

  const login = useCallback(
    async (payload: LoginPayload) => {
      const result = await authService.login(payload)
      setAuth(result.user, result.access_token)
      navigate(ROUTES.DASHBOARD, { replace: true })
    },
    [setAuth, navigate],
  )

  const register = useCallback(
    async (payload: RegisterPayload) => {
      await authService.register(payload)
      // Do not auto-login — user must verify their email first.
    },
    [],
  )

  const logout = useCallback(async () => {
    try {
      await authService.logout()
    } catch {
      // Silently swallowed — the session is cleared regardless of whether the
      // server-side logout succeeds (e.g. network failure).
    } finally {
      clearAuth()
      navigate(ROUTES.LOGIN, { replace: true })
    }
  }, [clearAuth, navigate])

  return {
    user,
    accessToken,
    isInitialised,
    isAuthenticated: !!accessToken,
    login,
    register,
    logout,
  }
}
