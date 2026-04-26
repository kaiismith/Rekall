import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { OrgSwitcher } from '@/components/common/ui/OrgSwitcher'
import { useOrgsStore } from '@/store/orgsStore'
import { useAuthStore } from '@/store/authStore'

const ORG = '00000000-0000-0000-0000-00000000a001'

const mockNavigate = vi.fn()
vi.mock('react-router-dom', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router-dom')>()
  return { ...actual, useNavigate: () => mockNavigate }
})

function wrap(initialPath = '/dashboard') {
  return render(
    <ThemeProvider theme={theme}>
      <MemoryRouter initialEntries={[initialPath]}>
        {/* Route declarations so useParams() inside OrgSwitcher resolves
            ":id" / ":orgId" against the test URL — without a matching Route
            the params object is empty and the trigger falls back to "Personal". */}
        <Routes>
          <Route path="/organizations/:id/*" element={<OrgSwitcher />} />
          <Route path="/organizations/:id" element={<OrgSwitcher />} />
          <Route path="*" element={<OrgSwitcher />} />
        </Routes>
      </MemoryRouter>
    </ThemeProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  useAuthStore.setState({ user: null, accessToken: null, isInitialised: true })
  useOrgsStore.setState({ orgs: [], isLoading: false, error: null })
})

describe('OrgSwitcher', () => {
  it('shows "Personal" when not on a scoped route and no current org', () => {
    wrap()
    expect(screen.getByRole('button', { name: /personal/i })).toBeInTheDocument()
  })

  it('shows the org name when on a scoped route', () => {
    useOrgsStore.setState({
      orgs: [
        {
          id: ORG,
          name: 'Acme',
          slug: 'acme',
          owner_id: 'u',
          created_at: '2026-01-01T00:00:00Z',
          updated_at: '2026-01-01T00:00:00Z',
        },
      ],
      isLoading: false,
      error: null,
    })
    wrap(`/organizations/${ORG}/meetings`)
    // The trigger renders the org name inside an inner span; find by text
    // (accessible-name role lookup matches "Personal" via the trigger button
    // when a chevron icon is present, so we go by visible text instead).
    expect(screen.getByText('Acme')).toBeInTheDocument()
  })

  it('opens the menu and navigates to /dashboard when "Personal" entry is selected', () => {
    wrap()
    fireEvent.click(screen.getByRole('button'))
    fireEvent.click(screen.getByText(/open meetings and calls/i)) // Personal MenuItem
    expect(mockNavigate).toHaveBeenCalledWith('/dashboard')
  })

  it('zero-org + non-admin: shows "Contact your administrator" disabled item', () => {
    wrap()
    fireEvent.click(screen.getByRole('button'))
    expect(screen.getByText(/contact your administrator/i)).toBeInTheDocument()
  })

  it('zero-org + platform admin: shows clickable "Create your first organization"', () => {
    useAuthStore.setState({
      user: {
        id: 'u',
        email: 'a@a',
        full_name: 'A',
        role: 'admin',
        email_verified: true,
        created_at: '2026-01-01T00:00:00Z',
      },
      accessToken: null,
      isInitialised: true,
    })
    wrap()
    fireEvent.click(screen.getByRole('button'))
    fireEvent.click(screen.getByText(/create your first organization/i))
    expect(mockNavigate).toHaveBeenCalledWith('/organizations')
  })
})
