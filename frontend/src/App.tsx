import { useMemo } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { ThemeProvider, createTheme } from '@mui/material/styles'
import CssBaseline from '@mui/material/CssBaseline'
import baseTheme from './theme'
import { Layout } from '@/components/layout/Layout'
import { ErrorBoundary } from '@/components/common/ErrorBoundary'
import { ProtectedRoute } from '@/components/common/ProtectedRoute'
import { useBootstrap } from '@/hooks/useBootstrap'
import { useUIPreferencesStore } from '@/store/uiPreferencesStore'
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
import { DeptDetailPage } from '@/pages/DeptDetailPage'
import { ScopedMeetingsPage } from '@/pages/ScopedMeetingsPage'
import { ScopedCallsPage } from '@/pages/ScopedCallsPage'
import { InviteAcceptPage } from '@/pages/InviteAcceptPage'
import { MeetingsPage } from '@/pages/MeetingsPage'
import { RecordsPage } from '@/pages/RecordsPage'
import { NewMeetingPage } from '@/pages/NewMeetingPage'
import { MeetingRoomPage } from '@/pages/MeetingRoomPage'
import { ProfilePage } from '@/pages/ProfilePage'
import { SettingsPage } from '@/pages/SettingsPage'
import { HelpPage } from '@/pages/HelpPage'
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
      <Route
        element={
          <ProtectedRoute>
            <Layout />
          </ProtectedRoute>
        }
      >
        <Route path={ROUTES.DASHBOARD} element={<DashboardPage />} />
        <Route path={ROUTES.CALLS} element={<CallsPage />} />
        <Route path={ROUTES.ORGANIZATIONS} element={<OrganizationsPage />} />
        <Route path={ROUTES.ORG_DETAIL} element={<OrgDetailPage />} />
        <Route path={ROUTES.ORG_MEETINGS} element={<ScopedMeetingsPage />} />
        <Route path={ROUTES.ORG_CALLS} element={<ScopedCallsPage />} />
        <Route path={ROUTES.ORG_DEPT_DETAIL} element={<DeptDetailPage />} />
        <Route path={ROUTES.ORG_DEPT_MEETINGS} element={<ScopedMeetingsPage />} />
        <Route path={ROUTES.ORG_DEPT_CALLS} element={<ScopedCallsPage />} />
        {/* Records uses a two-pane layout: the list always renders on the
            left; the right pane is the empty state on /records, or the
            selected record's detail on /records/:code. Both routes point to
            the same component which reads :code from useParams. */}
        <Route path={ROUTES.RECORDS} element={<RecordsPage />} />
        <Route path={ROUTES.RECORD_DETAIL} element={<RecordsPage />} />
        <Route path={ROUTES.MEETINGS} element={<MeetingsPage />} />
        <Route path={ROUTES.PROFILE} element={<ProfilePage />} />
        <Route path={ROUTES.SETTINGS} element={<SettingsPage />} />
        <Route path={ROUTES.HELP} element={<HelpPage />} />
      </Route>

      {/* Rekall landing — full-screen, no sidebar chrome, still requires auth.
          /meetings/new is the canonical create-meeting flow; /records/new is
          an alias from the Records tab so users on that page can start one too. */}
      <Route
        path={ROUTES.NEW_MEETING}
        element={
          <ProtectedRoute>
            <NewMeetingPage />
          </ProtectedRoute>
        }
      />
      <Route
        path={ROUTES.NEW_RECORD}
        element={
          <ProtectedRoute>
            <NewMeetingPage />
          </ProtectedRoute>
        }
      />

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

/**
 * Wraps MUI's ThemeProvider with a reduced-motion override read from the
 * UI preferences store. When the user opts in, MUI transitions are
 * short-circuited globally without rebuilding the base theme.
 */
function ThemedApp({ children }: { children: React.ReactNode }) {
  const reducedMotion = useUIPreferencesStore((s) => s.reducedMotion)

  const theme = useMemo(() => {
    if (!reducedMotion) return baseTheme
    return createTheme(baseTheme, {
      transitions: {
        create: () => 'none',
        duration: {
          shortest: 0,
          shorter: 0,
          short: 0,
          standard: 0,
          complex: 0,
          enteringScreen: 0,
          leavingScreen: 0,
        },
      },
    })
  }, [reducedMotion])

  return <ThemeProvider theme={theme}>{children}</ThemeProvider>
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemedApp>
        <CssBaseline />
        <ErrorBoundary>
          <BrowserRouter>
            <AppRoutes />
          </BrowserRouter>
        </ErrorBoundary>
      </ThemedApp>
      {import.meta.env.DEV && <ReactQueryDevtools initialIsOpen={false} />}
    </QueryClientProvider>
  )
}
