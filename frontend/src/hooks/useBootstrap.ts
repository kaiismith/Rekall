import { useEffect } from 'react'
import { useAuthStore } from '@/store/authStore'
import { authService } from '@/services/authService'
import { ApiError } from '@/services/api'

let bootstrapPromise: Promise<void> | null = null

function isExpectedAuthFailure(err: unknown): boolean {
  if (err instanceof ApiError) {
    return err.status === 401 || err.code === 'NETWORK_ERROR' || err.code === 'TIMEOUT'
  }
  return false
}

async function runBootstrap(): Promise<void> {
  const store = useAuthStore.getState()

  try {
    if (store.accessToken) {
      const user = await authService.me()
      store.setAuth(user, store.accessToken)
      return
    }

    const { access_token } = await authService.refresh()
    // Install the token BEFORE calling /me — the axios request interceptor
    // reads the store at request time, so the token must be in place or the
    // /me call goes out without an Authorization header and 401s.
    useAuthStore.setState({ accessToken: access_token })
    try {
      const user = await authService.me()
      store.setAuth(user, access_token)
    } catch (err) {
      // Refresh worked but /me failed (transient). Keep the token; pages that
      // need `user` can handle the null case with their own loading state.
      console.warn('[bootstrap] refresh succeeded but /me failed:', err)
    }
  } catch (err: unknown) {
    if (!isExpectedAuthFailure(err)) {
      console.warn('[bootstrap]', err)
    }
  } finally {
    useAuthStore.getState().setInitialised()
  }
}

/**
 * Runs bootstrap once per app session. Guarded at the module level so React
 * StrictMode's double-invocation in dev does not produce two network calls.
 * The promise is cleared when it settles so tests that reset the auth store
 * can trigger a fresh bootstrap on the next mount.
 */
export function useBootstrap() {
  useEffect(() => {
    if (useAuthStore.getState().isInitialised) return
    if (bootstrapPromise) return
    bootstrapPromise = runBootstrap().finally(() => {
      bootstrapPromise = null
    })
  }, [])
}
