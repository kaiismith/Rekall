import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { ScopePicker } from '@/components/common/ui/ScopePicker'
import { useOrgsStore } from '@/store/orgsStore'
import { useDeptsStore } from '@/store/deptsStore'
import type { Scope } from '@/types/scope'

const ORG = '00000000-0000-0000-0000-00000000a001'
const DEPT = '00000000-0000-0000-0000-00000000d001'

function wrap(ui: React.ReactElement) {
  return render(
    <ThemeProvider theme={theme}>
      <MemoryRouter>{ui}</MemoryRouter>
    </ThemeProvider>,
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

describe('ScopePicker', () => {
  it('shows the default "All scopes" label when value is null', () => {
    wrap(<ScopePicker value={null} onChange={vi.fn()} />)
    expect(screen.getByText('All scopes')).toBeInTheDocument()
  })

  it('shows "Personal (open)" when the open scope is selected', () => {
    wrap(<ScopePicker value={{ type: 'open' }} onChange={vi.fn()} />)
    expect(screen.getByText('Personal (open)')).toBeInTheDocument()
  })

  it('shows the org name when an org scope is selected', () => {
    wrap(<ScopePicker value={{ type: 'organization', id: ORG }} onChange={vi.fn()} />)
    expect(screen.getByText('Acme')).toBeInTheDocument()
  })

  it('shows "Org › Dept" when a dept scope is selected', () => {
    wrap(
      <ScopePicker
        value={{ type: 'department', id: DEPT, orgId: ORG }}
        onChange={vi.fn()}
      />,
    )
    expect(screen.getByText('Acme › Engineering')).toBeInTheDocument()
  })

  it('opens the popover and selects "Personal (open)"', () => {
    const onChange = vi.fn()
    wrap(<ScopePicker value={null} onChange={onChange} />)
    fireEvent.click(screen.getByText('All scopes'))
    fireEvent.click(screen.getByText('Personal (open)'))
    expect(onChange).toHaveBeenCalledWith({ type: 'open' })
  })

  it('selects an organization entry from the menu', () => {
    const onChange = vi.fn()
    wrap(<ScopePicker value={null} onChange={onChange} />)
    fireEvent.click(screen.getByText('All scopes'))
    // The org name appears in the popover list as well as the chip area;
    // grab the menu's instance via the list item's role.
    const acmeRow = screen.getAllByText('Acme')[0]!
    fireEvent.click(acmeRow)
    expect(onChange).toHaveBeenCalledWith({ type: 'organization', id: ORG })
  })

  it('clears the filter when the chip delete icon is clicked (value present)', () => {
    const onChange = vi.fn()
    const value: Scope = { type: 'organization', id: ORG }
    const { container } = wrap(<ScopePicker value={value} onChange={onChange} />)
    // MUI Chip's delete button has aria-label "delete" (or role button with class)
    const deleteBtn = container.querySelector('.MuiChip-deleteIcon')
    expect(deleteBtn).not.toBeNull()
    fireEvent.click(deleteBtn!)
    expect(onChange).toHaveBeenCalledWith(null)
  })
})
