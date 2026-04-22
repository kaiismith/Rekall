import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { ThemeProvider } from '@mui/material/styles'
import CssBaseline from '@mui/material/CssBaseline'
import theme from './theme'
import { Layout } from '@/components/layout/Layout'
import { ErrorBoundary } from '@/components/common/ErrorBoundary'
import { ProtectedRoute } from '@/components/common/ProtectedRoute'
import { useBootstrap } from '@/hooks/useBootstrap'
import { DashboardPage } from '@/pages/DashboardPage'
import { CallsPage } from '@/pages/CallsPage'
import { NotFoundPage } from '@/pages/NotFoundPage'
import { LoginPage } from '@/pages/LoginPage'
import { RegisterPage } from '@/pages/RegisterPage'
import { ForgotPasswordPage } from '@/pages/ForgotPasswordPage'
import { ResetPasswordPage } from '@/pages/ResetPasswordPage'
import { VerifyEmailPage } from '@/pages/VerifyEmailPage'
import { OrganizationsPage } from '@/pages/OrganizationsPage'
import { OrgDetailPage } from '@/pages/OrgDetailPage'
import { InviteAcceptPage } from '@/pages/InviteAcceptPage'
import { MeetingsPage } from '@/pages/MeetingsPage'
import { NewMeetingPage } from '@/pages/NewMeetingPage'
import { MeetingRoomPage } from '@/pages/MeetingRoomPage'
import { ROUTES } from '@/constants'

// Import authStore to wire the Axios interceptors for token injection + refresh.
import '@/store/authStore'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
      refetchOnWindowFocus: false,
    },
    mutations: {
      retry: 0,
    },
  },
})

function AppRoutes() {
  useBootstrap()

  return (
    <Routes>
      {/* Redirect root to dashboard */}
      <Route path={ROUTES.ROOT} element={<Navigate to={ROUTES.DASHBOARD} replace />} />

      {/* Public auth pages */}
      <Route path={ROUTES.LOGIN} element={<LoginPage />} />
      <Route path={ROUTES.REGISTER} element={<RegisterPage />} />
      <Route path={ROUTES.FORGOT_PASSWORD} element={<ForgotPasswordPage />} />
      <Route path={ROUTES.RESET_PASSWORD} element={<ResetPasswordPage />} />
      <Route path={ROUTES.VERIFY_EMAIL} element={<VerifyEmailPage />} />

      {/* Invitation accept (auth-aware: redirects to login if not signed in) */}
      <Route path={ROUTES.INVITATION_ACCEPT} element={<InviteAcceptPage />} />

      {/* Authenticated shell — ProtectedRoute guards, Layout provides the chrome */}
      <Route element={<ProtectedRoute><Layout /></ProtectedRoute>}>
        <Route path={ROUTES.DASHBOARD} element={<DashboardPage />} />
        <Route path={ROUTES.CALLS} element={<CallsPage />} />
        <Route path={ROUTES.ORGANIZATIONS} element={<OrganizationsPage />} />
        <Route path={ROUTES.ORG_DETAIL} element={<OrgDetailPage />} />
        <Route path={ROUTES.MEETINGS} element={<MeetingsPage />} />
        <Route path={ROUTES.NEW_MEETING} element={<NewMeetingPage />} />
      </Route>

      {/* Meeting room — full-screen, no sidebar chrome, still requires auth */}
      <Route
        path={ROUTES.MEETING_ROOM}
        element={
          <ProtectedRoute>
            <MeetingRoomPage />
          </ProtectedRoute>
        }
      />

      {/* 404 */}
      <Route path={ROUTES.NOT_FOUND} element={<NotFoundPage />} />
    </Routes>
  )
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        <ErrorBoundary>
          <BrowserRouter>
            <AppRoutes />
          </BrowserRouter>
        </ErrorBoundary>
      </ThemeProvider>
      {import.meta.env.DEV && <ReactQueryDevtools initialIsOpen={false} />}
    </QueryClientProvider>
  )
}
