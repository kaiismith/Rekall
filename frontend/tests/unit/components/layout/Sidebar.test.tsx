import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import { MemoryRouter } from 'react-router-dom'
import theme from '@/theme'
import { Sidebar, __resetSidebarSeedForTests } from '@/components/layout/Sidebar'
import { useUIStore } from '@/store/uiStore'
import { useUIPreferencesStore } from '@/store/uiPreferencesStore'

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
    // Reset stores to a known state before each test.
    useUIStore.setState({ sidebarOpen: true })
    useUIPreferencesStore.setState({ sidebarDefault: 'expanded' })
    __resetSidebarSeedForTests()
  })

  // ── open state ───────────────────────────────────────────────────────────────
  it('shows the "Rekall" brand text when open', () => {
    renderSidebar()
    expect(screen.getByText('Rekall')).toBeInTheDocument()
  })

  it('shows nav item labels when open', () => {
    renderSidebar()
    expect(screen.getByText('Dashboard')).toBeInTheDocument()
    expect(screen.getByText('Records')).toBeInTheDocument()
    expect(screen.getByText('Organizations')).toBeInTheDocument()
    // "Calls" and "Meetings" entries removed; the Records tab is the single
    // entry point for past sessions.
    expect(screen.queryByText('Calls')).not.toBeInTheDocument()
    expect(screen.queryByText('Meetings')).not.toBeInTheDocument()
  })

  // ── collapsed state ──────────────────────────────────────────────────────────
  it('hides "Rekall" brand text when collapsed', () => {
    useUIPreferencesStore.setState({ sidebarDefault: 'collapsed' })
    useUIStore.setState({ sidebarOpen: false })
    renderSidebar()
    expect(screen.queryByText('Rekall')).not.toBeInTheDocument()
  })

  it('hides nav item labels when collapsed', () => {
    useUIPreferencesStore.setState({ sidebarDefault: 'collapsed' })
    useUIStore.setState({ sidebarOpen: false })
    renderSidebar()
    expect(screen.queryByText('Dashboard')).not.toBeInTheDocument()
    expect(screen.queryByText('Records')).not.toBeInTheDocument()
  })

  // ── navigation ────────────────────────────────────────────────────────────────
  it('renders the Rekall wordmark when open and the "R" glyph when collapsed', () => {
    // Open: full wordmark + tagline.
    renderSidebar()
    expect(screen.getByText('Rekall')).toBeInTheDocument()
    expect(screen.getByText(/intelligence platform/i)).toBeInTheDocument()

    // Collapsed: single "R" glyph in the tile.
    useUIPreferencesStore.setState({ sidebarDefault: 'collapsed' })
    useUIStore.setState({ sidebarOpen: false })
    __resetSidebarSeedForTests()
    const { unmount } = renderSidebar()
    expect(screen.getAllByText('R').length).toBeGreaterThanOrEqual(1)
    unmount()
  })

  it('renders a nav button for each nav item', () => {
    renderSidebar()
    const buttons = screen.getAllByRole('button')
    // Primary nav (Dashboard, Records, Organizations) + footer (Profile, Settings) = 5
    expect(buttons.length).toBeGreaterThanOrEqual(5)
  })

  it('renders Profile and Settings in the footer', () => {
    renderSidebar()
    expect(screen.getByText('Profile')).toBeInTheDocument()
    expect(screen.getByText('Settings')).toBeInTheDocument()
  })

  it('marks Profile as selected when on /profile', () => {
    renderSidebar('/profile')
    const buttons = screen.getAllByRole('button')
    const profileBtn = buttons.find((b) => b.textContent?.includes('Profile'))
    expect(profileBtn?.classList.contains('Mui-selected')).toBe(true)
  })

  it('marks Settings as selected when on /settings', () => {
    renderSidebar('/settings')
    const buttons = screen.getAllByRole('button')
    const settingsBtn = buttons.find((b) => b.textContent?.includes('Settings'))
    expect(settingsBtn?.classList.contains('Mui-selected')).toBe(true)
  })

  // ── active state ──────────────────────────────────────────────────────────────
  it('marks the Dashboard button as selected on the dashboard path', () => {
    renderSidebar('/dashboard')
    // MUI ListItemButton marks active items with the "Mui-selected" CSS class
    const buttons = screen.getAllByRole('button')
    const dashBtn = buttons.find((b) => b.textContent?.includes('Dashboard'))
    expect(dashBtn?.classList.contains('Mui-selected')).toBe(true)
  })

  it('marks a nested path as active (e.g. /records/abc activates Records)', () => {
    renderSidebar('/records/abc-defg-hij')
    const buttons = screen.getAllByRole('button')
    const recBtn = buttons.find((b) => b.textContent?.includes('Records'))
    expect(recBtn?.classList.contains('Mui-selected')).toBe(true)
  })

  it('does not mark Dashboard as selected on /records path', () => {
    renderSidebar('/records')
    const buttons = screen.getAllByRole('button')
    const dashBtn = buttons.find((b) => b.textContent?.includes('Dashboard'))
    expect(dashBtn?.classList.contains('Mui-selected')).toBe(false)
  })
})
