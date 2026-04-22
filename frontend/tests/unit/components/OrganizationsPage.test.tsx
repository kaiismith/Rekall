import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { ThemeProvider } from '@mui/material/styles'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import theme from '@/theme'
import { OrganizationsPage } from '@/pages/OrganizationsPage'
import { useAuthStore } from '@/store/authStore'
import * as orgServiceModule from '@/services/organizationService'
import { ApiError } from '@/services/api'

vi.mock('@/services/organizationService', () => ({
  organizationService: {
    list: vi.fn(),
    create: vi.fn(),
    get: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    listMembers: vi.fn(),
    updateMember: vi.fn(),
    removeMember: vi.fn(),
    inviteUser: vi.fn(),
    acceptInvitation: vi.fn(),
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

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <ThemeProvider theme={theme}>
        <MemoryRouter initialEntries={['/organizations']}>
          <Routes>
            <Route path="/organizations" element={<OrganizationsPage />} />
            <Route path="/organizations/:id" element={<div>Org Detail</div>} />
          </Routes>
        </MemoryRouter>
      </ThemeProvider>
    </QueryClientProvider>,
  )
}

describe('OrganizationsPage', () => {
  beforeEach(() => {
    useAuthStore.setState({ user: mockUser, accessToken: 'token', isInitialised: true })
    vi.clearAllMocks()
  })

  it('renders page title and new organization button', () => {
    vi.mocked(orgServiceModule.organizationService.list).mockResolvedValue([])

    renderPage()

    expect(screen.getByText('Organizations')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /new organization/i })).toBeInTheDocument()
  })

  it('shows empty state when user has no organizations', async () => {
    vi.mocked(orgServiceModule.organizationService.list).mockResolvedValue([])

    renderPage()

    await waitFor(() => {
      expect(screen.getByText(/don't belong to any organizations/i)).toBeInTheDocument()
    })
  })

  it('renders organization cards from the API', async () => {
    vi.mocked(orgServiceModule.organizationService.list).mockResolvedValue([
      { id: 'org-1', name: 'Acme Corp', slug: 'acme-corp', owner_id: '1', created_at: '', updated_at: '' },
      { id: 'org-2', name: 'Globex', slug: 'globex', owner_id: '1', created_at: '', updated_at: '' },
    ])

    renderPage()

    await waitFor(() => {
      expect(screen.getByText('Acme Corp')).toBeInTheDocument()
      expect(screen.getByText('Globex')).toBeInTheDocument()
    })
  })

  it('opens create dialog when button is clicked', async () => {
    vi.mocked(orgServiceModule.organizationService.list).mockResolvedValue([])

    renderPage()

    fireEvent.click(screen.getByRole('button', { name: /new organization/i }))

    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByLabelText(/organization name/i)).toBeInTheDocument()
  })

  it('create button is disabled when name is empty', async () => {
    vi.mocked(orgServiceModule.organizationService.list).mockResolvedValue([])

    renderPage()

    fireEvent.click(screen.getByRole('button', { name: /new organization/i }))

    const createBtn = screen.getByRole('button', { name: /^create$/i })
    expect(createBtn).toBeDisabled()
  })

  it('creates organization and navigates to detail on success', async () => {
    vi.mocked(orgServiceModule.organizationService.list).mockResolvedValue([])
    vi.mocked(orgServiceModule.organizationService.create).mockResolvedValue({
      id: 'new-org', name: 'New Corp', slug: 'new-corp', owner_id: '1', created_at: '', updated_at: '',
    })

    renderPage()

    fireEvent.click(screen.getByRole('button', { name: /new organization/i }))
    fireEvent.change(screen.getByLabelText(/organization name/i), { target: { value: 'New Corp' } })
    fireEvent.click(screen.getByRole('button', { name: /^create$/i }))

    await waitFor(() => {
      expect(screen.getByText('Org Detail')).toBeInTheDocument()
    })
  })

  it('shows error message when create fails', async () => {
    vi.mocked(orgServiceModule.organizationService.list).mockResolvedValue([])
    vi.mocked(orgServiceModule.organizationService.create).mockRejectedValue(
      new ApiError('UNPROCESSABLE', 'organization name is required', 422),
    )

    renderPage()

    fireEvent.click(screen.getByRole('button', { name: /new organization/i }))
    fireEvent.change(screen.getByLabelText(/organization name/i), { target: { value: 'x' } })
    fireEvent.click(screen.getByRole('button', { name: /^create$/i }))

    await waitFor(() => {
      expect(screen.getByText('organization name is required')).toBeInTheDocument()
    })
  })

  it('shows error alert when list API fails', async () => {
    vi.mocked(orgServiceModule.organizationService.list).mockRejectedValue(
      new ApiError('INTERNAL_ERROR', 'service unavailable', 500),
    )

    renderPage()

    await waitFor(() => {
      expect(screen.getByRole('alert')).toBeInTheDocument()
    })
  })
})
