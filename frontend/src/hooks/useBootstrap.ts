import { useEffect } from 'react'
import { useAuthStore } from '@/store/authStore'
import { authService } from '@/services/authService'

/**
 * Called once at app startup to restore the auth session.
 * Tries GET /auth/me (which automatically retries with refresh token via the interceptor).
 * Always marks the store as initialised when done so ProtectedRoute can render.
 */
export function useBootstrap() {
  const { setAuth, setInitialised, accessToken } = useAuthStore()

  useEffect(() => {
    if (accessToken) {
      // Already have a token in memory — fetch the current user to confirm session.
      authService
        .me()
        .then((user) => setAuth(user, accessToken))
        .catch(() => {
          /* interceptor will clear auth on 401 */
        })
        .finally(() => setInitialised())
      return
    }

    // No in-memory token — attempt a silent refresh using the HttpOnly cookie.
    authService
      .refresh()
      .then(({ access_token }) =>
        authService.me().then((user) => setAuth(user, access_token)),
      )
      .catch(() => {
        /* No valid session — that's fine */
      })
      .finally(() => setInitialised())
  }, []) // eslint-disable-line react-hooks/exhaustive-deps
}
