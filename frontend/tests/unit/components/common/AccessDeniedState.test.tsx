import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { AccessDeniedState } from '@/components/common/ui/AccessDeniedState'

const mockNavigate = vi.fn()
vi.mock('react-router-dom', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router-dom')>()
  return { ...actual, useNavigate: () => mockNavigate }
})

function renderState() {
  return render(
    <ThemeProvider theme={theme}>
      <MemoryRouter>
        <AccessDeniedState />
      </MemoryRouter>
    </ThemeProvider>,
  )
}

describe('AccessDeniedState', () => {
  beforeEach(() => vi.clearAllMocks())

  it('renders the heading and explanation copy', () => {
    renderState()
    expect(
      screen.getByRole('heading', { name: /you don't have access to this space/i }),
    ).toBeInTheDocument()
    expect(screen.getByText(/private to its members/i)).toBeInTheDocument()
  })

  it('navigates to /dashboard when "Back to workspace" is clicked', () => {
    renderState()
    fireEvent.click(screen.getByRole('button', { name: /back to workspace/i }))
    expect(mockNavigate).toHaveBeenCalledWith('/dashboard')
  })
})
