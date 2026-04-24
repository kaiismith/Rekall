import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { TopBar } from '@/components/layout/TopBar'
import { useUIStore } from '@/store/uiStore'
import { useAuthStore } from '@/store/authStore'

function renderTopBar() {
  return render(
    <MemoryRouter>
      <ThemeProvider theme={theme}>
        <TopBar />
      </ThemeProvider>
    </MemoryRouter>,
  )
}

describe('TopBar', () => {
  beforeEach(() => {
    useUIStore.setState({ sidebarOpen: true })
    useAuthStore.setState({
      user: {
        id: 'user-1',
        email: 'travis@techvify.io',
        full_name: 'Travis Duong',
        role: 'member',
        email_verified: true,
        created_at: '2024-01-01T00:00:00Z',
      },
      accessToken: 'tok',
      isInitialised: true,
    })
  })

  it('renders the toggle sidebar button', () => {
    renderTopBar()
    expect(screen.getByRole('button', { name: /toggle sidebar/i })).toBeInTheDocument()
  })

  it('keeps the sidebar open state when rendered with sidebarOpen=true', () => {
    renderTopBar()
    expect(useUIStore.getState().sidebarOpen).toBe(true)
  })

  it('reflects the collapsed sidebar state', () => {
    useUIStore.setState({ sidebarOpen: false })
    renderTopBar()
    expect(screen.getByRole('button', { name: /toggle sidebar/i })).toBeInTheDocument()
    expect(useUIStore.getState().sidebarOpen).toBe(false)
  })

  it('clicking the toggle button flips the sidebar state', () => {
    renderTopBar()
    expect(useUIStore.getState().sidebarOpen).toBe(true)
    fireEvent.click(screen.getByRole('button', { name: /toggle sidebar/i }))
    expect(useUIStore.getState().sidebarOpen).toBe(false)
  })

  it('clicking twice returns the sidebar to open state', () => {
    renderTopBar()
    const btn = screen.getByRole('button', { name: /toggle sidebar/i })
    fireEvent.click(btn)
    fireEvent.click(btn)
    expect(useUIStore.getState().sidebarOpen).toBe(true)
  })

  it('renders the Notifications, Help, and Account buttons', () => {
    renderTopBar()
    expect(screen.getByRole('button', { name: /notifications/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /help/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /account/i })).toBeInTheDocument()
  })

  it('shows the user\'s initials in the avatar', () => {
    renderTopBar()
    // Travis Duong → "TD"
    expect(screen.getByText('TD')).toBeInTheDocument()
  })

  it('opens the account menu with the identity block and a sign-out option', () => {
    renderTopBar()
    fireEvent.click(screen.getByRole('button', { name: /account/i }))
    expect(screen.getByText('Travis Duong')).toBeInTheDocument()
    expect(screen.getByText('travis@techvify.io')).toBeInTheDocument()
    expect(screen.getByRole('menuitem', { name: /sign out/i })).toBeInTheDocument()
    // Profile and Settings moved to the sidebar — they should NOT be in the menu.
    expect(screen.queryByRole('menuitem', { name: /profile/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('menuitem', { name: /settings/i })).not.toBeInTheDocument()
  })
})
