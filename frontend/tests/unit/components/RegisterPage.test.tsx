import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { RegisterPage } from '@/pages/RegisterPage'
import * as authServiceModule from '@/services/authService'
import { ApiError } from '@/services/api'

vi.mock('@/services/authService', () => ({
  authService: {
    verifyEmail: vi.fn(),
    register: vi.fn(),
    login: vi.fn(),
    logout: vi.fn(),
    refresh: vi.fn(),
    me: vi.fn(),
    resendVerification: vi.fn(),
    forgotPassword: vi.fn(),
    resetPassword: vi.fn(),
  },
}))

function renderPage() {
  return render(
    <ThemeProvider theme={theme}>
      <MemoryRouter initialEntries={['/register']}>
        <Routes>
          <Route path="/register" element={<RegisterPage />} />
          <Route path="/login" element={<div>Login Page</div>} />
        </Routes>
      </MemoryRouter>
    </ThemeProvider>,
  )
}

describe('RegisterPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  // ── Initial render ────────────────────────────────────────────────────────────

  it('renders the create account heading', () => {
    renderPage()

    expect(screen.getByRole('heading', { name: /create your account/i })).toBeInTheDocument()
  })

  it('renders full name, email, and password fields', () => {
    renderPage()

    expect(screen.getByLabelText(/full name/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/email/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^password/i)).toBeInTheDocument()
  })

  it('renders the create account submit button', () => {
    renderPage()

    expect(screen.getByRole('button', { name: /create account/i })).toBeInTheDocument()
  })

  it('renders a sign in link', () => {
    renderPage()

    expect(screen.getByRole('link', { name: /sign in/i })).toBeInTheDocument()
  })

  // ── Submission ────────────────────────────────────────────────────────────────

  it('calls register with full name, email, and password on submit', async () => {
    vi.mocked(authServiceModule.authService.register).mockResolvedValue({
      id: '1', email: 'alice@example.com', full_name: 'Alice', role: 'member',
      email_verified: false, created_at: '',
    })

    renderPage()

    fireEvent.change(screen.getByLabelText(/full name/i), { target: { value: 'Alice' } })
    fireEvent.change(screen.getByLabelText(/email/i), { target: { value: 'alice@example.com' } })
    fireEvent.change(screen.getByLabelText(/^password/i), { target: { value: 'Password1!' } })
    fireEvent.click(screen.getByRole('button', { name: /create account/i }))

    await waitFor(() => {
      expect(authServiceModule.authService.register).toHaveBeenCalledWith({
        full_name: 'Alice',
        email: 'alice@example.com',
        password: 'Password1!',
      })
    })
  })

  // ── Success state ─────────────────────────────────────────────────────────────

  it('shows check-inbox screen after successful registration', async () => {
    vi.mocked(authServiceModule.authService.register).mockResolvedValue({
      id: '1', email: 'alice@example.com', full_name: 'Alice', role: 'member',
      email_verified: false, created_at: '',
    })

    renderPage()

    fireEvent.change(screen.getByLabelText(/full name/i), { target: { value: 'Alice' } })
    fireEvent.change(screen.getByLabelText(/email/i), { target: { value: 'alice@example.com' } })
    fireEvent.change(screen.getByLabelText(/^password/i), { target: { value: 'Password1!' } })
    fireEvent.click(screen.getByRole('button', { name: /create account/i }))

    await waitFor(() => {
      expect(screen.getByText(/check your inbox/i)).toBeInTheDocument()
    })
  })

  it('includes the submitted email in the success message', async () => {
    vi.mocked(authServiceModule.authService.register).mockResolvedValue({
      id: '1', email: 'alice@example.com', full_name: 'Alice', role: 'member',
      email_verified: false, created_at: '',
    })

    renderPage()

    fireEvent.change(screen.getByLabelText(/full name/i), { target: { value: 'Alice' } })
    fireEvent.change(screen.getByLabelText(/email/i), { target: { value: 'alice@example.com' } })
    fireEvent.change(screen.getByLabelText(/^password/i), { target: { value: 'Password1!' } })
    fireEvent.click(screen.getByRole('button', { name: /create account/i }))

    await waitFor(() => {
      expect(screen.getByText('alice@example.com')).toBeInTheDocument()
    })
  })

  it('shows back to sign in link on the success screen', async () => {
    vi.mocked(authServiceModule.authService.register).mockResolvedValue({
      id: '1', email: 'alice@example.com', full_name: 'Alice', role: 'member',
      email_verified: false, created_at: '',
    })

    renderPage()

    fireEvent.change(screen.getByLabelText(/full name/i), { target: { value: 'Alice' } })
    fireEvent.change(screen.getByLabelText(/email/i), { target: { value: 'alice@example.com' } })
    fireEvent.change(screen.getByLabelText(/^password/i), { target: { value: 'Password1!' } })
    fireEvent.click(screen.getByRole('button', { name: /create account/i }))

    await waitFor(() => {
      expect(screen.getByRole('link', { name: /back to sign in/i })).toBeInTheDocument()
    })
  })

  // ── Error handling ────────────────────────────────────────────────────────────

  it('shows ApiError message when registration fails', async () => {
    vi.mocked(authServiceModule.authService.register).mockRejectedValue(
      new ApiError('CONFLICT', 'email already in use', 409),
    )

    renderPage()

    fireEvent.change(screen.getByLabelText(/full name/i), { target: { value: 'Alice' } })
    fireEvent.change(screen.getByLabelText(/email/i), { target: { value: 'alice@example.com' } })
    fireEvent.change(screen.getByLabelText(/^password/i), { target: { value: 'Password1!' } })
    fireEvent.click(screen.getByRole('button', { name: /create account/i }))

    await waitFor(() => {
      expect(screen.getByRole('alert')).toBeInTheDocument()
      expect(screen.getByText('email already in use')).toBeInTheDocument()
    })
  })

  it('shows generic error message for non-ApiError failures', async () => {
    vi.mocked(authServiceModule.authService.register).mockRejectedValue(
      new Error('Network timeout'),
    )

    renderPage()

    fireEvent.change(screen.getByLabelText(/full name/i), { target: { value: 'Alice' } })
    fireEvent.change(screen.getByLabelText(/email/i), { target: { value: 'alice@example.com' } })
    fireEvent.change(screen.getByLabelText(/^password/i), { target: { value: 'Password1!' } })
    fireEvent.click(screen.getByRole('button', { name: /create account/i }))

    await waitFor(() => {
      expect(screen.getByText(/unexpected error/i)).toBeInTheDocument()
    })
  })

  it('keeps the form visible after an error', async () => {
    vi.mocked(authServiceModule.authService.register).mockRejectedValue(
      new ApiError('CONFLICT', 'email already in use', 409),
    )

    renderPage()

    fireEvent.change(screen.getByLabelText(/full name/i), { target: { value: 'Alice' } })
    fireEvent.change(screen.getByLabelText(/email/i), { target: { value: 'alice@example.com' } })
    fireEvent.change(screen.getByLabelText(/^password/i), { target: { value: 'Password1!' } })
    fireEvent.click(screen.getByRole('button', { name: /create account/i }))

    await waitFor(() => {
      expect(screen.getByLabelText(/email/i)).toBeInTheDocument()
    })
  })

  // ── Loading state ─────────────────────────────────────────────────────────────

  it('disables the button and shows Creating account… while request is in flight', async () => {
    let resolve!: (v: ReturnType<typeof authServiceModule.authService.register> extends Promise<infer T> ? T : never) => void
    vi.mocked(authServiceModule.authService.register).mockReturnValue(
      new Promise((r) => { resolve = r }),
    )

    renderPage()

    fireEvent.change(screen.getByLabelText(/full name/i), { target: { value: 'Alice' } })
    fireEvent.change(screen.getByLabelText(/email/i), { target: { value: 'alice@example.com' } })
    fireEvent.change(screen.getByLabelText(/^password/i), { target: { value: 'Password1!' } })
    fireEvent.click(screen.getByRole('button', { name: /create account/i }))

    expect(screen.getByRole('button', { name: /creating account/i })).toBeDisabled()

    resolve({ id: '1', email: 'alice@example.com', full_name: 'Alice', role: 'member', email_verified: false, created_at: '' })
  })
})
