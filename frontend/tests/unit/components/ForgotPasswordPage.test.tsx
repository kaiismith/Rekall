import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { ForgotPasswordPage } from '@/pages/ForgotPasswordPage'
import * as authServiceModule from '@/services/authService'

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
      <MemoryRouter initialEntries={['/forgot-password']}>
        <Routes>
          <Route path="/forgot-password" element={<ForgotPasswordPage />} />
          <Route path="/login" element={<div>Login Page</div>} />
        </Routes>
      </MemoryRouter>
    </ThemeProvider>,
  )
}

describe('ForgotPasswordPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  // ── Initial render ────────────────────────────────────────────────────────────

  it('renders the reset password heading', () => {
    renderPage()

    expect(screen.getByRole('heading', { name: /reset your password/i })).toBeInTheDocument()
  })

  it('renders an email input field', () => {
    renderPage()

    expect(screen.getByLabelText(/email/i)).toBeInTheDocument()
  })

  it('renders the send reset link button', () => {
    renderPage()

    expect(screen.getByRole('button', { name: /send reset link/i })).toBeInTheDocument()
  })

  it('renders a back to sign in link', () => {
    renderPage()

    expect(screen.getByRole('link', { name: /back to sign in/i })).toBeInTheDocument()
  })

  // ── Submission ────────────────────────────────────────────────────────────────

  it('calls forgotPassword with the entered email', async () => {
    vi.mocked(authServiceModule.authService.forgotPassword).mockResolvedValue()

    renderPage()

    fireEvent.change(screen.getByLabelText(/email/i), { target: { value: 'alice@example.com' } })
    fireEvent.click(screen.getByRole('button', { name: /send reset link/i }))

    await waitFor(() => {
      expect(authServiceModule.authService.forgotPassword).toHaveBeenCalledWith({ email: 'alice@example.com' })
    })
  })

  // ── Confirmation state (anti-enumeration) ─────────────────────────────────────

  it('shows confirmation screen after submission regardless of outcome (success)', async () => {
    vi.mocked(authServiceModule.authService.forgotPassword).mockResolvedValue()

    renderPage()

    fireEvent.change(screen.getByLabelText(/email/i), { target: { value: 'alice@example.com' } })
    fireEvent.click(screen.getByRole('button', { name: /send reset link/i }))

    await waitFor(() => {
      expect(screen.getByText(/check your inbox/i)).toBeInTheDocument()
      expect(screen.getByText(/if an account with that email exists/i)).toBeInTheDocument()
    })
  })

  it('shows confirmation screen even when the API rejects (anti-enumeration)', async () => {
    vi.mocked(authServiceModule.authService.forgotPassword).mockImplementation(
      () => new Promise((_, reject) => { reject(new Error('Network error')) }),
    )

    renderPage()

    fireEvent.change(screen.getByLabelText(/email/i), { target: { value: 'unknown@example.com' } })
    fireEvent.click(screen.getByRole('button', { name: /send reset link/i }))

    await waitFor(() => {
      expect(screen.getByText(/check your inbox/i)).toBeInTheDocument()
    })
  })

  it('shows Back to sign in button on confirmation screen', async () => {
    vi.mocked(authServiceModule.authService.forgotPassword).mockResolvedValue()

    renderPage()

    fireEvent.change(screen.getByLabelText(/email/i), { target: { value: 'alice@example.com' } })
    fireEvent.click(screen.getByRole('button', { name: /send reset link/i }))

    await waitFor(() => {
      // On confirmation screen the link becomes a button (outlined variant)
      expect(screen.getByRole('link', { name: /back to sign in/i })).toBeInTheDocument()
    })
  })

  // ── Loading state ─────────────────────────────────────────────────────────────

  it('disables the button and shows Sending… while request is in flight', async () => {
    let resolve!: () => void
    vi.mocked(authServiceModule.authService.forgotPassword).mockReturnValue(
      new Promise<void>((r) => { resolve = r }),
    )

    renderPage()

    fireEvent.change(screen.getByLabelText(/email/i), { target: { value: 'alice@example.com' } })
    fireEvent.click(screen.getByRole('button', { name: /send reset link/i }))

    expect(screen.getByRole('button', { name: /sending/i })).toBeDisabled()

    resolve()
  })
})
