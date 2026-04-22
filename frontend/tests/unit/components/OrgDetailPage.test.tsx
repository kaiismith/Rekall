import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { ThemeProvider } from '@mui/material/styles'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import theme from '@/theme'
import { OrgDetailPage } from '@/pages/OrgDetailPage'
import { useAuthStore } from '@/store/authStore'
import * as orgServiceModule from '@/services/organizationService'
import { ApiError } from '@/services/api'

vi.mock('@/services/organizationService', () => ({
  organizationService: {
    get: vi.fn(),
    list: vi.fn(),
    delete: vi.fn(),
    listMembers: vi.fn(),
    removeMember: vi.fn(),
    inviteUser: vi.fn(),
    listDepartments: vi.fn(),
    createDepartment: vi.fn(),
    deleteDepartment: vi.fn(),
    listDeptMembers: vi.fn(),
    addDeptMember: vi.fn(),
    removeDeptMember: vi.fn(),
  },
}))

const mockOrg = {
  id: 'org-1',
  name: 'Acme Corp',
  slug: 'acme-corp',
  owner_id: 'user-owner',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
}

const mockOwnerMember = { user_id: 'user-owner', role: 'owner', joined_at: '2024-01-01T00:00:00Z' }
const mockAdminMember = { user_id: 'user-admin', role: 'admin', joined_at: '2024-01-01T00:00:00Z' }
const mockPlainMember = { user_id: 'user-plain', role: 'member', joined_at: '2024-01-01T00:00:00Z' }

const mockDepts = [
  { id: 'dept-1', org_id: 'org-1', name: 'Engineering', description: 'Build things', created_by: 'user-owner', created_at: '', updated_at: '' },
  { id: 'dept-2', org_id: 'org-1', name: 'Marketing', description: '', created_by: 'user-owner', created_at: '', updated_at: '' },
]

const mockDeptMembers = [
  { user_id: 'user-head', role: 'head', joined_at: '2024-01-01T00:00:00Z' },
  { user_id: 'user-plain', role: 'member', joined_at: '2024-01-02T00:00:00Z' },
]

function renderPage(userId = 'user-owner', orgId = 'org-1') {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <ThemeProvider theme={theme}>
        <MemoryRouter initialEntries={[`/organizations/${orgId}`]}>
          <Routes>
            <Route path="/organizations/:id" element={<OrgDetailPage />} />
            <Route path="/organizations" element={<div>Organizations List</div>} />
          </Routes>
        </MemoryRouter>
      </ThemeProvider>
    </QueryClientProvider>,
  )
}

describe('OrgDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // Default: current user is the owner
    useAuthStore.setState({ user: { id: 'user-owner', email: 'owner@example.com', full_name: 'Owner', role: 'member', email_verified: true, created_at: '' }, accessToken: 'token', isInitialised: true })
    vi.mocked(orgServiceModule.organizationService.listDepartments).mockResolvedValue([])
    vi.mocked(orgServiceModule.organizationService.listDeptMembers).mockResolvedValue([])
  })

  // ── Loading / error ──────────────────────────────────────────────────────

  it('shows loading spinner while org is fetching', () => {
    vi.mocked(orgServiceModule.organizationService.get).mockReturnValue(new Promise(() => {}))
    vi.mocked(orgServiceModule.organizationService.listMembers).mockReturnValue(new Promise(() => {}))

    renderPage()

    expect(screen.getByRole('progressbar')).toBeInTheDocument()
  })

  it('shows error alert when org fetch fails', async () => {
    vi.mocked(orgServiceModule.organizationService.get).mockRejectedValue(
      new ApiError('NOT_FOUND', 'Organization not found', 404),
    )
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([])

    renderPage()

    await waitFor(() => {
      expect(screen.getByRole('alert')).toBeInTheDocument()
      expect(screen.getByText('Organization not found')).toBeInTheDocument()
    })
  })

  // ── Org header ───────────────────────────────────────────────────────────

  it('renders org name and slug', async () => {
    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([mockOwnerMember])

    renderPage()

    await waitFor(() => {
      expect(screen.getByText('Acme Corp')).toBeInTheDocument()
      expect(screen.getByText('/acme-corp')).toBeInTheDocument()
    })
  })

  // ── Members table ────────────────────────────────────────────────────────

  it('renders members in the table', async () => {
    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([
      mockOwnerMember,
      mockAdminMember,
      mockPlainMember,
    ])

    renderPage()

    await waitFor(() => {
      expect(screen.getByText('user-owner')).toBeInTheDocument()
      expect(screen.getByText('user-admin')).toBeInTheDocument()
      expect(screen.getByText('user-plain')).toBeInTheDocument()
    })
  })

  it('shows invite button for org owner', async () => {
    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([mockOwnerMember])

    renderPage('user-owner')

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /invite member/i })).toBeInTheDocument()
    })
  })

  it('hides invite button for plain member', async () => {
    useAuthStore.setState({ user: { id: 'user-plain', email: 'plain@example.com', full_name: 'Plain', role: 'member', email_verified: true, created_at: '' }, accessToken: 'token', isInitialised: true })

    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([mockOwnerMember, mockPlainMember])

    renderPage('user-plain')

    await waitFor(() => {
      expect(screen.queryByRole('button', { name: /invite member/i })).not.toBeInTheDocument()
    })
  })

  // ── Invite dialog ────────────────────────────────────────────────────────

  it('opens invite dialog when button clicked', async () => {
    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([mockOwnerMember])

    renderPage()

    await waitFor(() => screen.getByRole('button', { name: /invite member/i }))
    fireEvent.click(screen.getByRole('button', { name: /invite member/i }))

    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByLabelText(/email address/i)).toBeInTheDocument()
  })

  it('send invitation button is disabled when email is empty', async () => {
    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([mockOwnerMember])

    renderPage()

    await waitFor(() => screen.getByRole('button', { name: /invite member/i }))
    fireEvent.click(screen.getByRole('button', { name: /invite member/i }))

    const sendBtn = screen.getByRole('button', { name: /send invitation/i })
    expect(sendBtn).toBeDisabled()
  })

  it('shows success alert after invitation is sent', async () => {
    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([mockOwnerMember])
    vi.mocked(orgServiceModule.organizationService.inviteUser).mockResolvedValue(undefined as never)

    renderPage()

    await waitFor(() => screen.getByRole('button', { name: /invite member/i }))
    fireEvent.click(screen.getByRole('button', { name: /invite member/i }))
    fireEvent.change(screen.getByLabelText(/email address/i), { target: { value: 'new@example.com' } })
    fireEvent.click(screen.getByRole('button', { name: /send invitation/i }))

    await waitFor(() => {
      expect(screen.getByText(/invitation sent/i)).toBeInTheDocument()
    })
  })

  it('shows error alert when invitation fails', async () => {
    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([mockOwnerMember])
    vi.mocked(orgServiceModule.organizationService.inviteUser).mockRejectedValue(
      new ApiError('CONFLICT', 'User is already a member', 409),
    )

    renderPage()

    await waitFor(() => screen.getByRole('button', { name: /invite member/i }))
    fireEvent.click(screen.getByRole('button', { name: /invite member/i }))
    fireEvent.change(screen.getByLabelText(/email address/i), { target: { value: 'existing@example.com' } })
    fireEvent.click(screen.getByRole('button', { name: /send invitation/i }))

    await waitFor(() => {
      expect(screen.getByText('User is already a member')).toBeInTheDocument()
    })
  })

  // ── Departments section ──────────────────────────────────────────────────

  it('shows empty departments message when there are none', async () => {
    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([mockOwnerMember])
    vi.mocked(orgServiceModule.organizationService.listDepartments).mockResolvedValue([])

    renderPage()

    await waitFor(() => {
      expect(screen.getByText(/no departments yet/i)).toBeInTheDocument()
    })
  })

  it('renders department cards', async () => {
    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([mockOwnerMember])
    vi.mocked(orgServiceModule.organizationService.listDepartments).mockResolvedValue(mockDepts)

    renderPage()

    await waitFor(() => {
      expect(screen.getByText('Engineering')).toBeInTheDocument()
      expect(screen.getByText('Marketing')).toBeInTheDocument()
    })
  })

  it('renders department description when present', async () => {
    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([mockOwnerMember])
    vi.mocked(orgServiceModule.organizationService.listDepartments).mockResolvedValue([mockDepts[0]])

    renderPage()

    await waitFor(() => {
      expect(screen.getByText('Build things')).toBeInTheDocument()
    })
  })

  it('shows New department button for admin', async () => {
    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([mockOwnerMember])

    renderPage()

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /new department/i })).toBeInTheDocument()
    })
  })

  it('hides New department button for plain member', async () => {
    useAuthStore.setState({ user: { id: 'user-plain', email: 'plain@example.com', full_name: 'Plain', role: 'member', email_verified: true, created_at: '' }, accessToken: 'token', isInitialised: true })
    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([mockOwnerMember, mockPlainMember])
    vi.mocked(orgServiceModule.organizationService.listDepartments).mockResolvedValue([])

    renderPage('user-plain')

    await waitFor(() => {
      expect(screen.queryByRole('button', { name: /new department/i })).not.toBeInTheDocument()
    })
  })

  // ── Create department dialog ─────────────────────────────────────────────

  it('opens create department dialog', async () => {
    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([mockOwnerMember])

    renderPage()

    await waitFor(() => screen.getByRole('button', { name: /new department/i }))
    fireEvent.click(screen.getByRole('button', { name: /new department/i }))

    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByLabelText(/department name/i)).toBeInTheDocument()
  })

  it('create department button disabled when name is empty', async () => {
    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([mockOwnerMember])

    renderPage()

    await waitFor(() => screen.getByRole('button', { name: /new department/i }))
    fireEvent.click(screen.getByRole('button', { name: /new department/i }))

    const createBtn = screen.getByRole('button', { name: /^create$/i })
    expect(createBtn).toBeDisabled()
  })

  it('creates department and closes dialog on success', async () => {
    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([mockOwnerMember])
    vi.mocked(orgServiceModule.organizationService.createDepartment).mockResolvedValue(mockDepts[0])

    renderPage()

    await waitFor(() => screen.getByRole('button', { name: /new department/i }))
    fireEvent.click(screen.getByRole('button', { name: /new department/i }))
    fireEvent.change(screen.getByLabelText(/department name/i), { target: { value: 'Engineering' } })
    fireEvent.click(screen.getByRole('button', { name: /^create$/i }))

    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })
  })

  it('shows error when create department fails', async () => {
    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([mockOwnerMember])
    vi.mocked(orgServiceModule.organizationService.createDepartment).mockRejectedValue(
      new ApiError('UNPROCESSABLE', 'department name already exists', 422),
    )

    renderPage()

    await waitFor(() => screen.getByRole('button', { name: /new department/i }))
    fireEvent.click(screen.getByRole('button', { name: /new department/i }))
    fireEvent.change(screen.getByLabelText(/department name/i), { target: { value: 'Dup' } })
    fireEvent.click(screen.getByRole('button', { name: /^create$/i }))

    await waitFor(() => {
      expect(screen.getByText('department name already exists')).toBeInTheDocument()
    })
  })

  // ── Department card expand / members ─────────────────────────────────────

  it('expands department card and shows members', async () => {
    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([mockOwnerMember])
    vi.mocked(orgServiceModule.organizationService.listDepartments).mockResolvedValue([mockDepts[0]])
    vi.mocked(orgServiceModule.organizationService.listDeptMembers).mockResolvedValue(mockDeptMembers)

    renderPage()

    await waitFor(() => screen.getByText('Engineering'))

    // Click the expand toggle button (last icon button in the card)
    const expandButtons = screen.getAllByRole('button')
    const toggleBtn = expandButtons.find((b) => b.querySelector('svg[data-testid="ExpandMoreIcon"]'))
    expect(toggleBtn).toBeTruthy()
    fireEvent.click(toggleBtn!)

    await waitFor(() => {
      expect(screen.getByText('user-head')).toBeInTheDocument()
      expect(screen.getByText('user-plain')).toBeInTheDocument()
    })
  })

  it('shows head chip when department has a head member', async () => {
    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([mockOwnerMember])
    vi.mocked(orgServiceModule.organizationService.listDepartments).mockResolvedValue([mockDepts[0]])
    vi.mocked(orgServiceModule.organizationService.listDeptMembers).mockResolvedValue([
      { user_id: 'user-head', role: 'head', joined_at: '2024-01-01T00:00:00Z' },
    ])

    renderPage()

    await waitFor(() => screen.getByText('Engineering'))

    // Expand the card
    const expandButtons = screen.getAllByRole('button')
    const toggleBtn = expandButtons.find((b) => b.querySelector('svg[data-testid="ExpandMoreIcon"]'))
    fireEvent.click(toggleBtn!)

    await waitFor(() => {
      expect(screen.getByText(/head:/i)).toBeInTheDocument()
    })
  })

  // ── Add dept member dialog ───────────────────────────────────────────────

  it('opens add dept member dialog when Add button clicked', async () => {
    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([mockOwnerMember])
    vi.mocked(orgServiceModule.organizationService.listDepartments).mockResolvedValue([mockDepts[0]])

    renderPage()

    await waitFor(() => screen.getByText('Engineering'))

    // The "Add" button on the department card
    const addBtn = screen.getByRole('button', { name: /^add$/i })
    fireEvent.click(addBtn)

    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByLabelText(/user id/i)).toBeInTheDocument()
  })

  // ── Danger zone ──────────────────────────────────────────────────────────

  it('shows danger zone for owner', async () => {
    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([mockOwnerMember])

    renderPage()

    await waitFor(() => {
      expect(screen.getByText(/danger zone/i)).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /delete organization/i })).toBeInTheDocument()
    })
  })

  it('hides danger zone for non-owner', async () => {
    useAuthStore.setState({ user: { id: 'user-admin', email: 'admin@example.com', full_name: 'Admin', role: 'member', email_verified: true, created_at: '' }, accessToken: 'token', isInitialised: true })
    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([mockOwnerMember, mockAdminMember])

    renderPage('user-admin')

    await waitFor(() => screen.getByText('Acme Corp'))

    expect(screen.queryByRole('button', { name: /delete organization/i })).not.toBeInTheDocument()
  })

  it('navigates to organizations list after successful delete', async () => {
    vi.stubGlobal('confirm', () => true)

    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([mockOwnerMember])
    vi.mocked(orgServiceModule.organizationService.delete).mockResolvedValue(undefined as never)

    renderPage()

    await waitFor(() => screen.getByRole('button', { name: /delete organization/i }))
    fireEvent.click(screen.getByRole('button', { name: /delete organization/i }))

    await waitFor(() => {
      expect(screen.getByText('Organizations List')).toBeInTheDocument()
    })

    vi.unstubAllGlobals()
  })

  // ── Members loading spinner ──────────────────────────────────────────────

  it('shows spinner while members are loading', async () => {
    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    // Keep members pending so membersLoading is true
    vi.mocked(orgServiceModule.organizationService.listMembers).mockReturnValue(new Promise(() => {}))

    renderPage()

    // Org loads, but members spinner should show (line 172)
    await waitFor(() => screen.getByText('Acme Corp'))
    // There should be at least one progressbar visible for the members section
    const spinners = screen.getAllByRole('progressbar')
    expect(spinners.length).toBeGreaterThanOrEqual(1)
  })

  // ── Delete department ───────────────────────────────────────────────────

  it('deletes department when confirm is accepted', async () => {
    vi.stubGlobal('confirm', () => true)
    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([mockOwnerMember])
    vi.mocked(orgServiceModule.organizationService.listDepartments).mockResolvedValue([mockDepts[0]])
    vi.mocked(orgServiceModule.organizationService.deleteDepartment).mockResolvedValue(undefined as never)

    renderPage()

    await waitFor(() => screen.getByText('Engineering'))

    // The dept delete button is an IconButton with a DeleteIcon inside.
    // Find all buttons with DeleteIcon and click the one on the dept card (not the org one).
    const allButtons = screen.getAllByRole('button')
    const deptDeleteBtn = allButtons.find(
      (b) => b.querySelector('svg[data-testid="DeleteIcon"]') && !b.textContent?.includes('organization'),
    )
    expect(deptDeleteBtn).toBeTruthy()
    fireEvent.click(deptDeleteBtn!)

    await waitFor(() => {
      expect(orgServiceModule.organizationService.deleteDepartment).toHaveBeenCalled()
    })

    vi.unstubAllGlobals()
  })

  // ── Add dept member error ───────────────────────────────────────────────

  it('shows error when add dept member fails', async () => {
    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([mockOwnerMember])
    vi.mocked(orgServiceModule.organizationService.listDepartments).mockResolvedValue([mockDepts[0]])
    vi.mocked(orgServiceModule.organizationService.addDeptMember).mockRejectedValue(
      new ApiError('CONFLICT', 'User already in department', 409),
    )

    renderPage()

    await waitFor(() => screen.getByText('Engineering'))

    // Open add member dialog
    const addBtn = screen.getByRole('button', { name: /^add$/i })
    fireEvent.click(addBtn)

    await waitFor(() => screen.getByRole('dialog'))

    // Fill in user ID and submit — button text is "Add" (not "Add member")
    fireEvent.change(screen.getByLabelText(/user id/i), { target: { value: 'user-new' } })
    // The submit button in the dialog says "Add" when idle
    const dialogActions = screen.getByRole('dialog').querySelectorAll('button')
    const submitBtn = Array.from(dialogActions).find((b) => b.textContent === 'Add')
    expect(submitBtn).toBeTruthy()
    fireEvent.click(submitBtn!)

    await waitFor(() => {
      expect(screen.getByText('User already in department')).toBeInTheDocument()
    })
  })

  it('does not delete when confirm is cancelled', async () => {
    vi.stubGlobal('confirm', () => false)

    vi.mocked(orgServiceModule.organizationService.get).mockResolvedValue(mockOrg)
    vi.mocked(orgServiceModule.organizationService.listMembers).mockResolvedValue([mockOwnerMember])
    vi.mocked(orgServiceModule.organizationService.delete).mockResolvedValue(undefined as never)

    renderPage()

    await waitFor(() => screen.getByRole('button', { name: /delete organization/i }))
    fireEvent.click(screen.getByRole('button', { name: /delete organization/i }))

    expect(orgServiceModule.organizationService.delete).not.toHaveBeenCalled()

    vi.unstubAllGlobals()
  })
})
