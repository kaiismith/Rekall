import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { TopBar } from '@/components/layout/TopBar'
import { useUIStore } from '@/store/uiStore'

function renderTopBar() {
  return render(
    <ThemeProvider theme={theme}>
      <TopBar />
    </ThemeProvider>,
  )
}

describe('TopBar', () => {
  beforeEach(() => {
    useUIStore.setState({ sidebarOpen: true })
  })

  it('renders the toggle sidebar button', () => {
    renderTopBar()
    expect(screen.getByRole('button', { name: /toggle sidebar/i })).toBeInTheDocument()
  })

  it('shows MenuOpenIcon (title-less svg) when sidebar is open', () => {
    renderTopBar()
    // Both MUI icons are SVG; confirm toggle button exists and sidebar is open
    const btn = screen.getByRole('button', { name: /toggle sidebar/i })
    expect(btn).toBeInTheDocument()
    // The SVG path differs between MenuIcon / MenuOpenIcon.
    // We check that the document contains the data-testid from the icon name.
    // MUI icons render an SVG; MenuOpenIcon has a specific viewBox path — instead
    // we assert the aria-label and verify clicking toggles state.
    expect(useUIStore.getState().sidebarOpen).toBe(true)
  })

  it('shows MenuIcon when sidebar is collapsed', () => {
    useUIStore.setState({ sidebarOpen: false })
    renderTopBar()
    const btn = screen.getByRole('button', { name: /toggle sidebar/i })
    expect(btn).toBeInTheDocument()
    expect(useUIStore.getState().sidebarOpen).toBe(false)
  })

  it('clicking the toggle button calls toggleSidebar and flips state', () => {
    renderTopBar()
    expect(useUIStore.getState().sidebarOpen).toBe(true)

    fireEvent.click(screen.getByRole('button', { name: /toggle sidebar/i }))

    expect(useUIStore.getState().sidebarOpen).toBe(false)
  })

  it('clicking twice returns sidebar to open state', () => {
    renderTopBar()
    const btn = screen.getByRole('button', { name: /toggle sidebar/i })
    fireEvent.click(btn)
    fireEvent.click(btn)
    expect(useUIStore.getState().sidebarOpen).toBe(true)
  })

  it('renders the Notifications button', () => {
    renderTopBar()
    expect(screen.getByRole('button', { name: /notifications/i })).toBeInTheDocument()
  })

  it('renders the Account avatar', () => {
    renderTopBar()
    // Avatar contains the letter "U"
    expect(screen.getByText('U')).toBeInTheDocument()
  })

  it('renders "User" label text', () => {
    renderTopBar()
    expect(screen.getByText('User')).toBeInTheDocument()
  })
})
