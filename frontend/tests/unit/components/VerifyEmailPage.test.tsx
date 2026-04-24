import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { VerifyEmailPage } from '@/pages/VerifyEmailPage'
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
      <MemoryRouter initialEntries={[`/verify${search}`]}>
        <Routes>
          <Route path="/verify" element={<VerifyEmailPage />} />
          <Route path="/login" element={<div>Login Page</div>} />
        </Routes>
      </MemoryRouter>
    </ThemeProvider>,
  )
}

describe('VerifyEmailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  // ── No token ─────────────────────────────────────────────────────────────────

  it('shows error immediately when no token is in the URL', () => {
    renderPage('')

    expect(screen.getByText(/verification failed/i)).toBeInTheDocument()
    expect(screen.getByText(/no verification token found/i)).toBeInTheDocument()
  })

  it('does not call verifyEmail when no token is present', () => {
    renderPage('')

    expect(authServiceModule.authService.verifyEmail).not.toHaveBeenCalled()
  })

  it('shows back-to-sign-in link on the no-token error state', () => {
    renderPage('')

    expect(screen.getByRole('link', { name: /back to sign in/i })).toBeInTheDocument()
  })

  // ── Loading state ─────────────────────────────────────────────────────────────

  it('shows loading spinner and message while verifying', () => {
    vi.mocked(authServiceModule.authService.verifyEmail).mockReturnValue(new Promise(() => {}))

    renderPage('valid-token')

    expect(screen.getByRole('progressbar')).toBeInTheDocument()
    expect(screen.getByText(/verifying your email/i)).toBeInTheDocument()
  })

  // ── Success state ─────────────────────────────────────────────────────────────

  it('shows success message after verification', async () => {
    vi.mocked(authServiceModule.authService.verifyEmail).mockResolvedValue()

    renderPage('valid-token')

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /email verified/i })).toBeInTheDocument()
      expect(screen.getAllByText(/confirmed/i).length).toBeGreaterThan(0)
    })
  })

  it('shows Sign in button on success', async () => {
    vi.mocked(authServiceModule.authService.verifyEmail).mockResolvedValue()

    renderPage('valid-token')

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /sign in/i })).toBeInTheDocument()
    })
  })

  it('calls verifyEmail with the token from the URL', async () => {
    vi.mocked(authServiceModule.authService.verifyEmail).mockResolvedValue()

    renderPage('my-verify-token')

    await waitFor(() => {
      expect(authServiceModule.authService.verifyEmail).toHaveBeenCalledWith('my-verify-token')
    })
  })

  // ── Error state ───────────────────────────────────────────────────────────────

  it('shows error message from ApiError when verification fails', async () => {
    vi.mocked(authServiceModule.authService.verifyEmail).mockRejectedValue(
      new ApiError('BAD_REQUEST', 'verification link is invalid or has expired', 400),
    )

    renderPage('bad-token')

    await waitFor(() => {
      expect(screen.getByText(/verification failed/i)).toBeInTheDocument()
      expect(screen.getByText('verification link is invalid or has expired')).toBeInTheDocument()
    })
  })

  it('shows the fallback message for non-ApiError failures', async () => {
    // Use mockImplementation with a deferred rejection to avoid the unhandled-rejection
    // warning that fires before useEffect can attach .catch() to the returned promise.
    vi.mocked(authServiceModule.authService.verifyEmail).mockImplementation(
      () => new Promise((_, reject) => { reject(new Error('Network error')) }),
    )

    renderPage('some-token')

    await waitFor(() => {
      expect(screen.getByText('Verification failed.')).toBeInTheDocument()
    })
  })

  it('shows back-to-sign-in link on error state', async () => {
    vi.mocked(authServiceModule.authService.verifyEmail).mockRejectedValue(
      new ApiError('BAD_REQUEST', 'expired', 400),
    )

    renderPage('expired-token')

    await waitFor(() => {
      expect(screen.getByRole('link', { name: /back to sign in/i })).toBeInTheDocument()
    })
  })
})
