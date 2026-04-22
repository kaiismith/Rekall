import { Navigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '@/store/authStore'
import { ROUTES } from '@/constants'

interface ProtectedRouteProps {
  children: React.ReactNode
}

/**
 * Wraps a route so only authenticated users can access it.
 * Unauthenticated users are redirected to /login with the intended path saved in state
 * so they can be forwarded there after a successful sign-in.
 */
export function ProtectedRoute({ children }: ProtectedRouteProps) {
  const { accessToken, isInitialised } = useAuthStore()
  const location = useLocation()

  // While the app is bootstrapping (checking session via /auth/me), render nothing.
  if (!isInitialised) return null

  if (!accessToken) {
    return <Navigate to={ROUTES.LOGIN} state={{ from: location }} replace />
  }

  return <>{children}</>
}
