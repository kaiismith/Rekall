import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { ThemeProvider } from '@mui/material/styles'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import theme from '@/theme'
import { OrgDetailPage } from '@/pages/OrgDetailPage'
import { useAuthStore } from '@/store/authStore'
import { useOrgsStore } from '@/store/orgsStore'
import { useDeptsStore } from '@/store/deptsStore'
import { ApiError } from '@/services/api'

// Stub every organizationService method so the page renders without network.
// Individual tests override `mockReturnValue` per case.
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

import * as orgServiceModule from '@/services/organizationService'
const orgService = orgServiceModule.organizationService as Record<
  keyof typeof orgServiceModule.organizationService,
  ReturnType<typeof vi.fn>
>

// ── Fixtures ──────────────────────────────────────────────────────────────────

const ORG_ID = '00000000-0000-0000-0000-00000000a001'
const DEPT_ID = '00000000-0000-0000-0000-00000000d001'

const mockOrg = {
  id: ORG_ID,
  name: 'Acme Corp',
  slug: 'acme-corp',
  owner_id: 'user-owner',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const ownerMember = {
  user_id: 'user-owner',
  org_id: ORG_ID,
  role: 'owner' as const,
  joined_at: '2026-01-01T00:00:00Z',
}
const plainMember = {
  user_id: 'user-plain',
  org_id: ORG_ID,
  role: 'member' as const,
  joined_at: '2026-01-01T00:00:00Z',
}

const mockDept = {
  id: DEPT_ID,
  org_id: ORG_ID,
  name: 'Engineering',
  description: 'Build things',
  created_by: 'user-owner',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

function renderPage(opts: { userId?: string; initialEntries?: string[] } = {}) {
  const userId = opts.userId ?? 'user-owner'
  useAuthStore.setState({
    user: {
      id: userId,
      email: `${userId}@x`,
      full_name: userId,
      role: 'member',
      email_verified: true,
      created_at: '2026-01-01T00:00:00Z',
    },
    accessToken: 'tok',
    isInitialised: true,
  })

  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <ThemeProvider theme={theme}>
        <MemoryRouter initialEntries={opts.initialEntries ?? [`/organizations/${ORG_ID}`]}>
          <Routes>
            <Route path="/organizations/:id" element={<OrgDetailPage />} />
            <Route path="/organizations" element={<div data-testid="orgs-list-page">Orgs</div>} />
            <Route
              path="/organizations/:orgId/departments/:deptId"
              element={<div data-testid="dept-detail-page">Dept</div>}
            />
          </Routes>
        </MemoryRouter>
      </ThemeProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  useOrgsStore.setState({ orgs: [], isLoading: false, error: null })
  useDeptsStore.setState({ byOrg: {}, isLoading: {}, errors: {} })

  orgService.get.mockResolvedValue(mockOrg)
  orgService.listMembers.mockResolvedValue([ownerMember])
  orgService.listDepartments.mockResolvedValue([mockDept])
})

// ─── Tabs ────────────────────────────────────────────────────────────────────

describe('OrgDetailPage — tabs', () => {
  it('renders the four tabs and starts on Overview', async () => {
    renderPage()
    await waitFor(() => expect(screen.getByText('Acme Corp')).toBeInTheDocument())
    expect(screen.getByRole('tab', { name: 'Overview', selected: true })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Departments' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Meetings' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Calls' })).toBeInTheDocument()
  })

  it('persists the selected tab via ?tab=', async () => {
    renderPage({ initialEntries: [`/organizations/${ORG_ID}?tab=departments`] })
    await waitFor(() =>
      expect(
        screen.getByRole('tab', { name: 'Departments', selected: true }),
      ).toBeInTheDocument(),
    )
    expect(await screen.findByText('Engineering')).toBeInTheDocument()
  })

  it('switches to Departments tab on click', async () => {
    renderPage()
    await waitFor(() => expect(screen.getByText('Acme Corp')).toBeInTheDocument())
    fireEvent.click(screen.getByRole('tab', { name: 'Departments' }))
    expect(await screen.findByText('Engineering')).toBeInTheDocument()
  })

  it('navigates to /organizations/:orgId/departments/:deptId when a dept card is clicked', async () => {
    renderPage({ initialEntries: [`/organizations/${ORG_ID}?tab=departments`] })
    const card = await screen.findByText('Engineering')
    fireEvent.click(card)
    expect(await screen.findByTestId('dept-detail-page')).toBeInTheDocument()
  })
})

// ─── Access denied ───────────────────────────────────────────────────────────

describe('OrgDetailPage — access', () => {
  it('renders AccessDeniedState on 403 from the org fetch', async () => {
    orgService.get.mockRejectedValue(new ApiError('FORBIDDEN', 'no', 403))
    renderPage()
    expect(
      await screen.findByRole('heading', { name: /you don't have access to this space/i }),
    ).toBeInTheDocument()
  })

  it('renders AccessDeniedState on 404 from the org fetch (does not leak existence)', async () => {
    orgService.get.mockRejectedValue(new ApiError('NOT_FOUND', 'gone', 404))
    renderPage()
    expect(
      await screen.findByRole('heading', { name: /you don't have access to this space/i }),
    ).toBeInTheDocument()
  })
})

// ─── Affordance gating ──────────────────────────────────────────────────────

describe('OrgDetailPage — overview affordance gating', () => {
  it('owner sees the "Delete organization" button', async () => {
    renderPage({ userId: 'user-owner' })
    expect(
      await screen.findByRole('button', { name: /delete organization/i }),
    ).toBeInTheDocument()
  })

  it('plain member does NOT see the danger zone', async () => {
    orgService.listMembers.mockResolvedValue([ownerMember, plainMember])
    renderPage({ userId: 'user-plain' })
    await waitFor(() => expect(screen.getByText('Acme Corp')).toBeInTheDocument())
    expect(screen.queryByRole('button', { name: /delete organization/i })).not.toBeInTheDocument()
  })

  it('owner sees the "Invite member" button', async () => {
    renderPage({ userId: 'user-owner' })
    expect(await screen.findByRole('button', { name: /invite member/i })).toBeInTheDocument()
  })

  it('plain member does NOT see "Invite member"', async () => {
    orgService.listMembers.mockResolvedValue([ownerMember, plainMember])
    renderPage({ userId: 'user-plain' })
    await waitFor(() => expect(screen.getByText('Acme Corp')).toBeInTheDocument())
    expect(screen.queryByRole('button', { name: /invite member/i })).not.toBeInTheDocument()
  })
})

describe('OrgDetailPage — departments affordance gating', () => {
  it('owner sees the "New department" button on the Departments tab', async () => {
    renderPage({
      userId: 'user-owner',
      initialEntries: [`/organizations/${ORG_ID}?tab=departments`],
    })
    expect(await screen.findByRole('button', { name: /new department/i })).toBeInTheDocument()
  })

  it('plain member does NOT see "New department"', async () => {
    orgService.listMembers.mockResolvedValue([ownerMember, plainMember])
    renderPage({
      userId: 'user-plain',
      initialEntries: [`/organizations/${ORG_ID}?tab=departments`],
    })
    await waitFor(() => expect(screen.getByText('Engineering')).toBeInTheDocument())
    expect(screen.queryByRole('button', { name: /new department/i })).not.toBeInTheDocument()
  })
})
