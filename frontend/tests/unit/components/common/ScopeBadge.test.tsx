import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { ScopeBadge } from '@/components/common/ui/ScopeBadge'
import { useOrgsStore } from '@/store/orgsStore'
import { useDeptsStore } from '@/store/deptsStore'

const ORG = '00000000-0000-0000-0000-00000000a001'
const DEPT = '00000000-0000-0000-0000-00000000d001'

const mockNavigate = vi.fn()
vi.mock('react-router-dom', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router-dom')>()
  return { ...actual, useNavigate: () => mockNavigate }
})

function wrapped(ui: React.ReactElement, initialPath = '/meetings') {
  return render(
    <ThemeProvider theme={theme}>
      <MemoryRouter initialEntries={[initialPath]}>{ui}</MemoryRouter>
    </ThemeProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  // Seed stores so the badge's lazy useEffect doesn't trigger fetches.
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
  useDeptsStore.setState({
    byOrg: {
      [ORG]: [
        {
          id: DEPT,
          org_id: ORG,
          name: 'Engineering',
          description: '',
          created_by: 'u',
          created_at: '2026-01-01T00:00:00Z',
          updated_at: '2026-01-01T00:00:00Z',
        },
      ],
    },
    isLoading: {},
    errors: {},
  })
})

describe('ScopeBadge', () => {
  it('renders "Open" for the open scope and is non-clickable', () => {
    wrapped(<ScopeBadge scope={{ type: 'open' }} />)
    const chip = screen.getByLabelText('Open')
    expect(chip).toBeInTheDocument()
    fireEvent.click(chip)
    expect(mockNavigate).not.toHaveBeenCalled()
  })

  it('renders the org name and navigates to the org meetings route on click', () => {
    wrapped(<ScopeBadge scope={{ type: 'organization', id: ORG }} />)
    const chip = screen.getByLabelText('Acme')
    fireEvent.click(chip)
    expect(mockNavigate).toHaveBeenCalledWith(`/organizations/${ORG}/meetings`)
  })

  it('renders "Org › Dept" and navigates to the dept meetings route on click', () => {
    wrapped(<ScopeBadge scope={{ type: 'department', id: DEPT, orgId: ORG }} />)
    const chip = screen.getByLabelText('Acme › Engineering')
    fireEvent.click(chip)
    expect(mockNavigate).toHaveBeenCalledWith(
      `/organizations/${ORG}/departments/${DEPT}/meetings`,
    )
  })

  it('is non-clickable on a scoped page (matchPath /organizations/:id/*)', () => {
    wrapped(
      <ScopeBadge scope={{ type: 'organization', id: ORG }} />,
      `/organizations/${ORG}/meetings`,
    )
    fireEvent.click(screen.getByLabelText('Acme'))
    expect(mockNavigate).not.toHaveBeenCalled()
  })

  it('honours forceNonClickable even on the flat list', () => {
    wrapped(<ScopeBadge scope={{ type: 'organization', id: ORG }} forceNonClickable />)
    fireEvent.click(screen.getByLabelText('Acme'))
    expect(mockNavigate).not.toHaveBeenCalled()
  })
})
