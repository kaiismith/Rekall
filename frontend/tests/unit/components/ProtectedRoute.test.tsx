import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { ThemeProvider } from '@mui/material/styles'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import theme from '@/theme'
import { ProtectedRoute } from '@/components/common/ProtectedRoute'
import { useAuthStore } from '@/store/authStore'

function renderWithProviders(ui: React.ReactElement, initialPath = '/protected') {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <ThemeProvider theme={theme}>
        <MemoryRouter initialEntries={[initialPath]}>
          {ui}
        </MemoryRouter>
      </ThemeProvider>
    </QueryClientProvider>,
  )
}

function App() {
  return (
    <Routes>
      <Route path="/login" element={<div>Login Page</div>} />
      <Route
        path="/protected"
        element={
          <ProtectedRoute>
            <div>Protected Content</div>
          </ProtectedRoute>
        }
      />
    </Routes>
  )
}

describe('ProtectedRoute', () => {
  beforeEach(() => {
    // Reset auth state before each test
    useAuthStore.setState({ user: null, accessToken: null, isInitialised: true })
  })

  it('renders children when user is authenticated', () => {
    useAuthStore.setState({
      accessToken: 'valid-token',
      user: { id: '1', email: 'a@b.com', full_name: 'Alice', role: 'member', email_verified: true, created_at: '' },
      isInitialised: true,
    })

    renderWithProviders(<App />)

    expect(screen.getByText('Protected Content')).toBeInTheDocument()
  })

  it('redirects to /login when not authenticated', () => {
    useAuthStore.setState({ accessToken: null, user: null, isInitialised: true })

    renderWithProviders(<App />)

    expect(screen.getByText('Login Page')).toBeInTheDocument()
    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument()
  })

  it('renders nothing while still initialising', () => {
    useAuthStore.setState({ accessToken: null, user: null, isInitialised: false })

    renderWithProviders(<App />)

    // Neither login nor protected content should show during bootstrap
    expect(screen.queryByText('Login Page')).not.toBeInTheDocument()
    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument()
  })
})
