import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { ThemeProvider } from '@mui/material/styles'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import theme from '@/theme'
import { LoginPage } from '@/pages/LoginPage'
import { RegisterPage } from '@/pages/RegisterPage'
import { useAuthStore } from '@/store/authStore'
import * as authServiceModule from '@/services/authService'

vi.mock('@/services/authService', () => ({
  authService: {
    login: vi.fn(),
    register: vi.fn(),
    logout: vi.fn(),
    refresh: vi.fn(),
    me: vi.fn(),
    verifyEmail: vi.fn(),
    resendVerification: vi.fn(),
    forgotPassword: vi.fn(),
    resetPassword: vi.fn(),
  },
}))

function renderPage(ui: React.ReactElement, initialPath = '/') {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <ThemeProvider theme={theme}>
        <MemoryRouter initialEntries={[initialPath]}>
          <Routes>
            <Route path="/" element={ui} />
            <Route path="/dashboard" element={<div>Dashboard</div>} />
            <Route path="/register" element={<RegisterPage />} />
            <Route path="/forgot-password" element={<div>Forgot Password</div>} />
          </Routes>
        </MemoryRouter>
      </ThemeProvider>
    </QueryClientProvider>,
  )
}

const mockUser = {
  id: '1', email: 'alice@example.com', full_name: 'Alice',
  role: 'member', email_verified: true, created_at: '',
}

describe('LoginPage', () => {
  beforeEach(() => {
    useAuthStore.setState({ user: null, accessToken: null, isInitialised: true })
    vi.clearAllMocks()
  })

  it('renders email and password fields', () => {
    renderPage(<LoginPage />)
    expect(screen.getByLabelText(/email/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^password/i)).toBeInTheDocument()
  })

  it('redirects to dashboard on successful login', async () => {
    vi.mocked(authServiceModule.authService.login).mockResolvedValue({
      access_token: 'tok', user: mockUser,
    })

    renderPage(<LoginPage />)

    fireEvent.change(screen.getByLabelText(/email/i), { target: { value: 'alice@example.com' } })
    fireEvent.change(screen.getByLabelText(/^password/i), { target: { value: 'password1' } })
    fireEvent.click(screen.getByRole('button', { name: /sign in/i }))

    await waitFor(() => {
      expect(screen.getByText('Dashboard')).toBeInTheDocument()
    })
  })

  it('shows error message on failed login', async () => {
    const { ApiError } = await import('@/services/api')
    vi.mocked(authServiceModule.authService.login).mockRejectedValue(
      new ApiError('UNAUTHORIZED', 'invalid email or password', 401),
    )

    renderPage(<LoginPage />)

    fireEvent.change(screen.getByLabelText(/email/i), { target: { value: 'alice@example.com' } })
    fireEvent.change(screen.getByLabelText(/^password/i), { target: { value: 'wrongpass' } })
    fireEvent.click(screen.getByRole('button', { name: /sign in/i }))

    await waitFor(() => {
      expect(screen.getByText('invalid email or password')).toBeInTheDocument()
    })
  })
})

describe('RegisterPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders registration form fields', () => {
    renderPage(<RegisterPage />)
    expect(screen.getByLabelText(/full name/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/email/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^password/i)).toBeInTheDocument()
  })

  it('shows success screen after successful registration', async () => {
    vi.mocked(authServiceModule.authService.register).mockResolvedValue(mockUser)

    renderPage(<RegisterPage />)

    fireEvent.change(screen.getByLabelText(/full name/i), { target: { value: 'Alice' } })
    fireEvent.change(screen.getByLabelText(/email/i), { target: { value: 'alice@example.com' } })
    fireEvent.change(screen.getByLabelText(/^password/i), { target: { value: 'password1' } })
    fireEvent.change(screen.getByLabelText(/confirm password/i), { target: { value: 'password1' } })
    fireEvent.click(screen.getByRole('button', { name: /create account/i }))

    await waitFor(() => {
      expect(screen.getByText('Check your inbox')).toBeInTheDocument()
    })
  })
})
