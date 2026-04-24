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
  it('renders the Error 404 eyebrow', () => {
    renderPage()
    expect(screen.getByText(/error 404/i)).toBeInTheDocument()
  })

  it('renders the "can\'t find that page" heading', () => {
    renderPage()
    expect(screen.getByRole('heading', { name: /can't find that page/i })).toBeInTheDocument()
  })

  it('renders the descriptive subtitle', () => {
    renderPage()
    expect(screen.getByText(/may have been moved, renamed, or no longer/i)).toBeInTheDocument()
  })

  it('renders "Go back" and "Go to dashboard" buttons', () => {
    renderPage()
    expect(screen.getByRole('button', { name: /go back/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /go to dashboard/i })).toBeInTheDocument()
  })

  it('navigates to dashboard when primary button is clicked', () => {
    renderPage()
    fireEvent.click(screen.getByRole('button', { name: /go to dashboard/i }))
    expect(mockNavigate).toHaveBeenCalledWith('/dashboard')
  })

  it('navigates back when secondary button is clicked', () => {
    renderPage()
    fireEvent.click(screen.getByRole('button', { name: /go back/i }))
    expect(mockNavigate).toHaveBeenCalledWith(-1)
  })
})
