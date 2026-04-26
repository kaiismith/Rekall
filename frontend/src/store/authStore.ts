import axios, { type InternalAxiosRequestConfig } from 'axios'
import { create } from 'zustand'
import { apiClient, ApiError } from '@/services/api'
import { authService } from '@/services/authService'
import type { User } from '@/types/auth'

interface AuthState {
  user: User | null
  accessToken: string | null
  isInitialised: boolean

  // Actions
  setAuth: (user: User, token: string) => void
  clearAuth: () => void
  setInitialised: () => void
}

export const useAuthStore = create<AuthState>()((set) => ({
  user: null,
  accessToken: null,
  isInitialised: false,

  setAuth: (user, accessToken) => set({ user, accessToken }),
  clearAuth: () => {
    set({ user: null, accessToken: null })
    // Drop the orgs/depts caches so the next sign-in does not reuse the
    // previous user's membership data. Done dynamically to avoid a circular
    // import (orgsStore is a sibling module that may be loaded before authStore).
    void import('@/store/orgsStore').then((m) => m.useOrgsStore.getState().invalidate())
  },
  setInitialised: () => set({ isInitialised: true }),
}))

// ── Axios interceptors (wired once at module load) ─────────────────────────────

/** Attach the current access token to every outgoing request. */
apiClient.interceptors.request.use((config) => {
  const token = useAuthStore.getState().accessToken
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

/** On 401, attempt a silent token refresh once, then retry. */
interface RetryableConfig extends InternalAxiosRequestConfig {
  _retried?: boolean
}

let isRefreshing = false
let pendingRequests: Array<(token: string) => void> = []

apiClient.interceptors.response.use(
  (response) => response,
  async (error: unknown) => {
    if (!axios.isAxiosError(error)) throw error

    const originalRequest = error.config as RetryableConfig | undefined
    if (!originalRequest) throw error

    // Only intercept 401s that haven't been retried yet,
    // and skip refresh/login/register endpoints to avoid infinite loops.
    const skipPaths = ['/auth/login', '/auth/register', '/auth/refresh']
    const isSkipped = skipPaths.some((p) => originalRequest.url?.includes(p))

    if (error.response?.status === 401 && !originalRequest._retried && !isSkipped) {
      if (isRefreshing) {
        // Queue this request until the in-flight refresh completes.
        return new Promise<string>((resolve) => {
          pendingRequests.push(resolve)
        }).then((newToken) => {
          originalRequest.headers.set('Authorization', `Bearer ${newToken}`)
          return apiClient(originalRequest)
        })
      }

      originalRequest._retried = true
      isRefreshing = true

      try {
        const { access_token } = await authService.refresh()
        const { user } = useAuthStore.getState()
        if (user) {
          useAuthStore.getState().setAuth(user, access_token)
        }

        // Flush queued requests
        pendingRequests.forEach((resolve) => resolve(access_token))
        pendingRequests = []

        originalRequest.headers.set('Authorization', `Bearer ${access_token}`)
        return apiClient(originalRequest)
      } catch {
        // Refresh failed — clear auth and let the caller handle the error.
        useAuthStore.getState().clearAuth()
        pendingRequests = []
        throw new ApiError('UNAUTHENTICATED', 'Your session has expired. Please sign in again.', 401)
      } finally {
        isRefreshing = false
      }
    }

    throw error
  },
)
