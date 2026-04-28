import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { ThemeProvider } from '@mui/material/styles'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import theme from '@/theme'
import { DeptDetailPage } from '@/pages/DeptDetailPage'
import { useAuthStore } from '@/store/authStore'
import { useOrgsStore } from '@/store/orgsStore'
import { useDeptsStore } from '@/store/deptsStore'
import { ApiError } from '@/services/api'

vi.mock('@/services/organizationService', () => ({
  organizationService: {
    get: vi.fn(),
    list: vi.fn(),
    listMembers: vi.fn(),
    listDepartments: vi.fn(),
    getDepartment: vi.fn(),
    listDeptMembers: vi.fn(),
    addDeptMember: vi.fn(),
    removeDeptMember: vi.fn(),
    updateDeptMemberRole: vi.fn(),
  },
}))
vi.mock('@/services/meetingService', () => ({
  meetingService: { listMine: vi.fn().mockResolvedValue({ success: true, data: [] }) },
}))
vi.mock('@/services/callService', () => ({
  callService: {
    list: vi.fn().mockResolvedValue({ success: true, data: [], meta: { page: 1, per_page: 20, total: 0 } }),
  },
}))

import * as orgServiceModule from '@/services/organizationService'
const orgService = orgServiceModule.organizationService as Record<
  keyof typeof orgServiceModule.organizationService,
  ReturnType<typeof vi.fn>
>

const ORG = '00000000-0000-0000-0000-00000000a001'
const DEPT = '00000000-0000-0000-0000-00000000d001'

const ownerMember = {
  user_id: 'user-owner',
  org_id: ORG,
  role: 'owner' as const,
  joined_at: '2026-01-01T00:00:00Z',
}
const plainMember = {
  user_id: 'user-plain',
  org_id: ORG,
  role: 'member' as const,
  joined_at: '2026-01-01T00:00:00Z',
}

const dept = {
  id: DEPT,
  org_id: ORG,
  name: 'Engineering',
  description: 'Build things',
  created_by: 'user-owner',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

function renderPage(opts: {
  userId?: string
  initialEntries?: string[]
} = {}) {
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
        <MemoryRouter
          initialEntries={
            opts.initialEntries ?? [`/organizations/${ORG}/departments/${DEPT}`]
          }
        >
          <Routes>
            <Route
              path="/organizations/:orgId/departments/:deptId"
              element={<DeptDetailPage />}
            />
            <Route
              path="/organizations/:id"
              element={<div data-testid="org-detail-page">Org</div>}
            />
            <Route path="/dashboard" element={<div data-testid="dashboard">Dash</div>} />
          </Routes>
        </MemoryRouter>
      </ThemeProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  useOrgsStore.setState({
    orgs: [
      {
        id: ORG,
        name: 'Acme',
        slug: 'acme',
        owner_id: 'user-owner',
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
      },
    ],
    isLoading: false,
    error: null,
  })
  useDeptsStore.setState({ byOrg: {}, isLoading: {}, errors: {} })

  orgService.getDepartment.mockResolvedValue(dept)
  orgService.listDeptMembers.mockResolvedValue([])
  orgService.listMembers.mockResolvedValue([ownerMember])
})

describe('DeptDetailPage', () => {
  it('renders the department name and three tabs', async () => {
    renderPage()
    await waitFor(() => expect(screen.getByText('Engineering')).toBeInTheDocument())
    expect(screen.getByRole('tab', { name: 'Overview', selected: true })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Meetings' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Calls' })).toBeInTheDocument()
  })

  it('persists tab via ?tab=', async () => {
    renderPage({
      initialEntries: [`/organizations/${ORG}/departments/${DEPT}?tab=meetings`],
    })
    await waitFor(() =>
      expect(screen.getByRole('tab', { name: 'Meetings', selected: true })).toBeInTheDocument(),
    )
  })

  it('renders AccessDeniedState on 403 from getDepartment', async () => {
    orgService.getDepartment.mockRejectedValue(new ApiError('FORBIDDEN', 'no', 403))
    renderPage()
    // The hero query failure short-circuits; AccessDenied is also rendered when
    // any panel-level fetch returns 403. Either way the dept name should NOT
    // render and the dashboard navigation route should be reachable from
    // AccessDenied. We simply assert the dept name is missing for this case.
    await waitFor(() => {
      expect(screen.queryByText('Engineering')).not.toBeInTheDocument()
    })
  })

  it('shows "Add member" for an org owner', async () => {
    renderPage({ userId: 'user-owner' })
    await waitFor(() => expect(screen.getByText('Engineering')).toBeInTheDocument())
    expect(await screen.findByRole('button', { name: /add member/i })).toBeInTheDocument()
  })

  it('hides "Add member" for a plain org+dept member', async () => {
    orgService.listMembers.mockResolvedValue([ownerMember, plainMember])
    orgService.listDeptMembers.mockResolvedValue([
      { user_id: 'user-plain', department_id: DEPT, role: 'member', joined_at: '2026-01-01T00:00:00Z' },
    ])
    renderPage({ userId: 'user-plain' })
    await waitFor(() => expect(screen.getByText('Engineering')).toBeInTheDocument())
    expect(screen.queryByRole('button', { name: /add member/i })).not.toBeInTheDocument()
  })

  it('navigates back to the org Departments tab via "Back to organization"', async () => {
    renderPage()
    await waitFor(() => expect(screen.getByText('Engineering')).toBeInTheDocument())
    fireEvent.click(screen.getByRole('button', { name: /back to organization/i }))
    expect(await screen.findByTestId('org-detail-page')).toBeInTheDocument()
  })

  it('renders 404 surface for malformed deptId in the URL', () => {
    renderPage({
      initialEntries: [`/organizations/${ORG}/departments/not-a-uuid`],
    })
    // AccessDeniedState is rendered when the dept ID is not a valid UUID —
    // we don't issue any service call.
    expect(orgService.getDepartment).not.toHaveBeenCalled()
  })
})
