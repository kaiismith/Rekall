import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { ResetPasswordPage } from '@/pages/ResetPasswordPage'
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

function renderPage(token = 'valid-token') {
  const search = token ? `?token=${token}` : ''
  return render(
    <ThemeProvider theme={theme}>
      <MemoryRouter initialEntries={[`/reset-password${search}`]}>
        <Routes>
          <Route path="/reset-password" element={<ResetPasswordPage />} />
          <Route path="/login" element={<div>Login Page</div>} />
          <Route path="/forgot-password" element={<div>Forgot Password Page</div>} />
        </Routes>
      </MemoryRouter>
    </ThemeProvider>,
  )
}

describe('ResetPasswordPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  // ── No token guard ────────────────────────────────────────────────────────────

  it('shows invalid-link error when no token is in the URL', () => {
    renderPage('')

    expect(screen.getByRole('alert')).toBeInTheDocument()
    expect(screen.getByText(/invalid or has expired/i)).toBeInTheDocument()
  })

  it('does not render the password form when token is missing', () => {
    renderPage('')

    expect(screen.queryByLabelText(/new password/i)).not.toBeInTheDocument()
  })

  it('shows a request new link on the no-token error screen', () => {
    renderPage('')

    expect(screen.getByRole('link', { name: /request a new reset link/i })).toBeInTheDocument()
  })

  it('does not call resetPassword when token is missing', () => {
    renderPage('')

    expect(authServiceModule.authService.resetPassword).not.toHaveBeenCalled()
  })

  // ── Form render (valid token) ─────────────────────────────────────────────────

  it('renders the set new password heading with a valid token', () => {
    renderPage('good-token')

    // heading and submit button share the same text — target the heading role
    expect(screen.getByRole('heading', { name: /set new password/i })).toBeInTheDocument()
  })

  it('renders a password input field', () => {
    renderPage('good-token')

    expect(screen.getByLabelText(/new password/i)).toBeInTheDocument()
  })

  it('renders the set new password submit button', () => {
    renderPage('good-token')

    expect(screen.getByRole('button', { name: /set new password/i })).toBeInTheDocument()
  })

  // ── Successful reset ──────────────────────────────────────────────────────────

  it('calls resetPassword with token and password on submit', async () => {
    vi.mocked(authServiceModule.authService.resetPassword).mockResolvedValue()

    renderPage('my-token')

    fireEvent.change(screen.getByLabelText(/new password/i), { target: { value: 'NewPass1!' } })
    fireEvent.click(screen.getByRole('button', { name: /set new password/i }))

    await waitFor(() => {
      expect(authServiceModule.authService.resetPassword).toHaveBeenCalledWith({
        token: 'my-token',
        password: 'NewPass1!',
      })
    })
  })

  it('navigates to login page after a successful reset', async () => {
    vi.mocked(authServiceModule.authService.resetPassword).mockResolvedValue()

    renderPage('my-token')

    fireEvent.change(screen.getByLabelText(/new password/i), { target: { value: 'NewPass1!' } })
    fireEvent.click(screen.getByRole('button', { name: /set new password/i }))

    await waitFor(() => {
      expect(screen.getByText('Login Page')).toBeInTheDocument()
    })
  })

  // ── Error handling ────────────────────────────────────────────────────────────

  it('shows ApiError message when reset fails', async () => {
    vi.mocked(authServiceModule.authService.resetPassword).mockRejectedValue(
      new ApiError('BAD_REQUEST', 'reset link is invalid or has expired', 400),
    )

    renderPage('expired-token')

    fireEvent.change(screen.getByLabelText(/new password/i), { target: { value: 'AnyPass1!' } })
    fireEvent.click(screen.getByRole('button', { name: /set new password/i }))

    await waitFor(() => {
      expect(screen.getByRole('alert')).toBeInTheDocument()
      expect(screen.getByText('reset link is invalid or has expired')).toBeInTheDocument()
    })
  })

  it('shows generic error message for non-ApiError failures', async () => {
    vi.mocked(authServiceModule.authService.resetPassword).mockRejectedValue(
      new Error('Network timeout'),
    )

    renderPage('some-token')

    fireEvent.change(screen.getByLabelText(/new password/i), { target: { value: 'AnyPass1!' } })
    fireEvent.click(screen.getByRole('button', { name: /set new password/i }))

    await waitFor(() => {
      expect(screen.getByText(/unexpected error/i)).toBeInTheDocument()
    })
  })

  it('keeps the form visible after an error (does not navigate away)', async () => {
    vi.mocked(authServiceModule.authService.resetPassword).mockRejectedValue(
      new ApiError('BAD_REQUEST', 'expired', 400),
    )

    renderPage('bad-token')

    fireEvent.change(screen.getByLabelText(/new password/i), { target: { value: 'AnyPass1!' } })
    fireEvent.click(screen.getByRole('button', { name: /set new password/i }))

    await waitFor(() => {
      expect(screen.getByLabelText(/new password/i)).toBeInTheDocument()
    })
  })

  // ── Loading state ─────────────────────────────────────────────────────────────

  it('disables the button and shows Saving… while request is in flight', async () => {
    let resolve!: () => void
    vi.mocked(authServiceModule.authService.resetPassword).mockReturnValue(
      new Promise<void>((r) => { resolve = r }),
    )

    renderPage('valid-token')

    fireEvent.change(screen.getByLabelText(/new password/i), { target: { value: 'Pass1!' } })
    fireEvent.click(screen.getByRole('button', { name: /set new password/i }))

    expect(screen.getByRole('button', { name: /saving/i })).toBeDisabled()

    resolve()
  })
})
