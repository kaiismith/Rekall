import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { InviteAcceptPage } from '@/pages/InviteAcceptPage'
import { useAuthStore } from '@/store/authStore'
import * as orgServiceModule from '@/services/organizationService'
import { ApiError } from '@/services/api'

vi.mock('@/services/organizationService', () => ({
  organizationService: {
    acceptInvitation: vi.fn(),
    list: vi.fn(),
    create: vi.fn(),
    get: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    listMembers: vi.fn(),
    updateMember: vi.fn(),
    removeMember: vi.fn(),
    inviteUser: vi.fn(),
    listDepartments: vi.fn(),
    createDepartment: vi.fn(),
    getDepartment: vi.fn(),
    updateDepartment: vi.fn(),
    deleteDepartment: vi.fn(),
    listDeptMembers: vi.fn(),
    addDeptMember: vi.fn(),
    updateDeptMember: vi.fn(),
    removeDeptMember: vi.fn(),
  },
}))

const mockUser = {
  id: '1', email: 'alice@example.com', full_name: 'Alice',
  role: 'member', email_verified: true, created_at: '',
}

function renderPage(token = 'valid-token', authenticated = true) {
  const search = token ? `?token=${token}` : ''
  return render(
    <ThemeProvider theme={theme}>
      <MemoryRouter initialEntries={[`/invitations/accept${search}`]}>
        <Routes>
          <Route path="/invitations/accept" element={<InviteAcceptPage />} />
          <Route path="/organizations" element={<div>Organizations</div>} />
          <Route path="/login" element={<div>Login Page</div>} />
          <Route path="/register" element={<div>Register Page</div>} />
          <Route path="/dashboard" element={<div>Dashboard</div>} />
        </Routes>
      </MemoryRouter>
    </ThemeProvider>,
  )
}

describe('InviteAcceptPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  // ── Unauthenticated state ────────────────────────────────────────────────────

  it('shows sign-in prompt when user is not authenticated', () => {
    useAuthStore.setState({ user: null, accessToken: null, isInitialised: true })

    renderPage('valid-token', false)

    expect(screen.getByRole('heading', { name: /sign in to accept/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /sign in/i })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /create an account/i })).toBeInTheDocument()
  })

  it('explains login requirement when unauthenticated', () => {
    useAuthStore.setState({ user: null, accessToken: null, isInitialised: true })

    renderPage('valid-token', false)

    expect(screen.getByText(/need to be signed in/i)).toBeInTheDocument()
  })

  // ── Loading state ────────────────────────────────────────────────────────────

  it('shows loading spinner while accepting invitation', () => {
    useAuthStore.setState({ user: mockUser, accessToken: 'token', isInitialised: true })
    vi.mocked(orgServiceModule.organizationService.acceptInvitation).mockReturnValue(new Promise(() => {}))

    renderPage('valid-token')

    expect(screen.getByRole('progressbar')).toBeInTheDocument()
    expect(screen.getByText(/accepting invitation/i)).toBeInTheDocument()
  })

  // ── Success state ────────────────────────────────────────────────────────────

  it('shows success message with org name after acceptance', async () => {
    useAuthStore.setState({ user: mockUser, accessToken: 'token', isInitialised: true })
    vi.mocked(orgServiceModule.organizationService.acceptInvitation).mockResolvedValue({
      id: 'org-1', name: 'Acme Corp', slug: 'acme-corp', owner_id: '1',
      created_at: '', updated_at: '',
    })

    renderPage('valid-token')

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /welcome aboard/i })).toBeInTheDocument()
      expect(screen.getAllByText(/Acme Corp/).length).toBeGreaterThan(0)
    })
  })

  it('shows Go to organizations button on success', async () => {
    useAuthStore.setState({ user: mockUser, accessToken: 'token', isInitialised: true })
    vi.mocked(orgServiceModule.organizationService.acceptInvitation).mockResolvedValue({
      id: 'org-1', name: 'Acme Corp', slug: 'acme-corp', owner_id: '1',
      created_at: '', updated_at: '',
    })

    renderPage('valid-token')

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /go to organizations/i })).toBeInTheDocument()
    })
  })

  // ── Error state ──────────────────────────────────────────────────────────────

  it('shows error message when invitation is invalid', async () => {
    useAuthStore.setState({ user: mockUser, accessToken: 'token', isInitialised: true })
    vi.mocked(orgServiceModule.organizationService.acceptInvitation).mockRejectedValue(
      new ApiError('BAD_REQUEST', 'invitation link is invalid, expired, or already accepted', 400),
    )

    renderPage('bad-token')

    await waitFor(() => {
      expect(screen.getByText(/invitation error/i)).toBeInTheDocument()
      expect(screen.getByText('invitation link is invalid, expired, or already accepted')).toBeInTheDocument()
    })
  })

  it('shows generic error message for non-ApiError failures', async () => {
    useAuthStore.setState({ user: mockUser, accessToken: 'token', isInitialised: true })
    vi.mocked(orgServiceModule.organizationService.acceptInvitation).mockRejectedValue(
      new Error('Network error'),
    )

    renderPage('some-token')

    await waitFor(() => {
      expect(screen.getByText(/failed to accept invitation/i)).toBeInTheDocument()
    })
  })

  it('shows Go to dashboard link on error', async () => {
    useAuthStore.setState({ user: mockUser, accessToken: 'token', isInitialised: true })
    vi.mocked(orgServiceModule.organizationService.acceptInvitation).mockRejectedValue(
      new ApiError('BAD_REQUEST', 'expired', 400),
    )

    renderPage('expired-token')

    await waitFor(() => {
      expect(screen.getByRole('link', { name: /go to dashboard/i })).toBeInTheDocument()
    })
  })

  // ── No token edge case ───────────────────────────────────────────────────────

  it('does not call acceptInvitation when no token is in the URL', () => {
    useAuthStore.setState({ user: mockUser, accessToken: 'token', isInitialised: true })

    renderPage('') // no token

    expect(orgServiceModule.organizationService.acceptInvitation).not.toHaveBeenCalled()
  })

  it('calls acceptInvitation with the token from the URL', async () => {
    useAuthStore.setState({ user: mockUser, accessToken: 'token', isInitialised: true })
    vi.mocked(orgServiceModule.organizationService.acceptInvitation).mockResolvedValue({
      id: 'org-1', name: 'Acme', slug: 'acme', owner_id: '1', created_at: '', updated_at: '',
    })

    renderPage('my-special-token')

    await waitFor(() => {
      expect(orgServiceModule.organizationService.acceptInvitation).toHaveBeenCalledWith({ token: 'my-special-token' })
    })
  })
})
