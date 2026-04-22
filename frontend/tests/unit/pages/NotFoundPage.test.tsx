import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { NotFoundPage } from '@/pages/NotFoundPage'

const mockNavigate = vi.fn()
vi.mock('react-router-dom', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router-dom')>()
  return { ...actual, useNavigate: () => mockNavigate }
})

function renderPage() {
  return render(
    <MemoryRouter>
      <ThemeProvider theme={theme}>
        <NotFoundPage />
      </ThemeProvider>
    </MemoryRouter>,
  )
}

describe('NotFoundPage', () => {
  it('renders the 404 heading', () => {
    renderPage()
    expect(screen.getAllByText('404').length).toBeGreaterThan(0)
  })

  it('renders the "Page not found" message', () => {
    renderPage()
    expect(screen.getByText('Page not found')).toBeInTheDocument()
  })

  it('renders the descriptive subtitle', () => {
    renderPage()
    expect(screen.getByText(/doesn't exist or has been moved/i)).toBeInTheDocument()
  })

  it('renders a "Back to Dashboard" button', () => {
    renderPage()
    expect(screen.getByRole('button', { name: /back to dashboard/i })).toBeInTheDocument()
  })

  it('navigates to dashboard when button is clicked', () => {
    renderPage()
    fireEvent.click(screen.getByRole('button', { name: /back to dashboard/i }))
    expect(mockNavigate).toHaveBeenCalledWith(expect.stringContaining('/'))
  })
})
