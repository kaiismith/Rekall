import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { DashboardPage } from '@/pages/DashboardPage'

function renderPage() {
  return render(
    <ThemeProvider theme={theme}>
      <DashboardPage />
    </ThemeProvider>,
  )
}

describe('DashboardPage', () => {
  it('renders the Dashboard heading', () => {
    renderPage()

    expect(screen.getByText('Dashboard')).toBeInTheDocument()
  })

  it('renders all four stat card labels', () => {
    renderPage()

    expect(screen.getByText(/total calls/i)).toBeInTheDocument()
    expect(screen.getByText(/completed/i)).toBeInTheDocument()
    expect(screen.getByText(/processing/i)).toBeInTheDocument()
    expect(screen.getByText(/avg duration/i)).toBeInTheDocument()
  })

  it('renders placeholder dashes for all stat values', () => {
    renderPage()

    const dashes = screen.getAllByText('—')
    expect(dashes.length).toBeGreaterThanOrEqual(4)
  })
})
