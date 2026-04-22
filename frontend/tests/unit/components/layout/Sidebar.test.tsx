import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import { MemoryRouter } from 'react-router-dom'
import theme from '@/theme'
import { Sidebar } from '@/components/layout/Sidebar'
import { useUIStore } from '@/store/uiStore'

// ── helpers ────────────────────────────────────────────────────────────────────

function renderSidebar(initialPath = '/') {
  return render(
    <ThemeProvider theme={theme}>
      <MemoryRouter initialEntries={[initialPath]}>
        <Sidebar />
      </MemoryRouter>
    </ThemeProvider>,
  )
}

// ── tests ──────────────────────────────────────────────────────────────────────

describe('Sidebar', () => {
  beforeEach(() => {
    // Reset store to open state before each test.
    useUIStore.setState({ sidebarOpen: true })
  })

  // ── open state ───────────────────────────────────────────────────────────────
  it('shows the "Rekall" brand text when open', () => {
    renderSidebar()
    expect(screen.getByText('Rekall')).toBeInTheDocument()
  })

  it('shows nav item labels when open', () => {
    renderSidebar()
    expect(screen.getByText('Dashboard')).toBeInTheDocument()
    expect(screen.getByText('Meetings')).toBeInTheDocument()
    expect(screen.getByText('Calls')).toBeInTheDocument()
    expect(screen.getByText('Organizations')).toBeInTheDocument()
  })

  // ── collapsed state ──────────────────────────────────────────────────────────
  it('hides "Rekall" brand text when collapsed', () => {
    useUIStore.setState({ sidebarOpen: false })
    renderSidebar()
    expect(screen.queryByText('Rekall')).not.toBeInTheDocument()
  })

  it('hides nav item labels when collapsed', () => {
    useUIStore.setState({ sidebarOpen: false })
    renderSidebar()
    expect(screen.queryByText('Dashboard')).not.toBeInTheDocument()
    expect(screen.queryByText('Meetings')).not.toBeInTheDocument()
  })

  // ── navigation ────────────────────────────────────────────────────────────────
  it('has the "R" logo box in both open and collapsed states', () => {
    renderSidebar()
    expect(screen.getByText('R')).toBeInTheDocument()
    useUIStore.setState({ sidebarOpen: false })
    // Re-render collapsed
    const { unmount } = renderSidebar()
    expect(screen.getAllByText('R').length).toBeGreaterThanOrEqual(1)
    unmount()
  })

  it('renders a nav button for each nav item', () => {
    renderSidebar()
    const buttons = screen.getAllByRole('button')
    // At least 4 nav item buttons (Dashboard, Calls, Meetings, Organizations)
    expect(buttons.length).toBeGreaterThanOrEqual(4)
  })

  // ── active state ──────────────────────────────────────────────────────────────
  it('marks the Dashboard button as selected on the dashboard path', () => {
    renderSidebar('/dashboard')
    // MUI ListItemButton marks active items with the "Mui-selected" CSS class
    const buttons = screen.getAllByRole('button')
    const dashBtn = buttons.find((b) => b.textContent?.includes('Dashboard'))
    expect(dashBtn?.classList.contains('Mui-selected')).toBe(true)
  })

  it('marks a nested path as active (e.g. /meetings/123 activates Meetings)', () => {
    renderSidebar('/meetings/abc-123')
    const buttons = screen.getAllByRole('button')
    const meetBtn = buttons.find((b) => b.textContent?.includes('Meetings'))
    expect(meetBtn?.classList.contains('Mui-selected')).toBe(true)
  })

  it('does not mark Dashboard as selected on /meetings path', () => {
    renderSidebar('/meetings')
    const buttons = screen.getAllByRole('button')
    const dashBtn = buttons.find((b) => b.textContent?.includes('Dashboard'))
    expect(dashBtn?.classList.contains('Mui-selected')).toBe(false)
  })
})
