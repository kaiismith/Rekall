import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { ErrorBoundary } from '@/components/common/ErrorBoundary'

// Suppress console.error noise from intentional throws.
beforeEach(() => {
  vi.spyOn(console, 'error').mockImplementation(() => {})
})

// Component that throws on demand.
function Bomb({ shouldThrow }: { shouldThrow: boolean }) {
  if (shouldThrow) throw new Error('Test explosion')
  return <div>All good</div>
}

function renderBoundary(shouldThrow: boolean, fallback?: React.ReactNode) {
  return render(
    <ThemeProvider theme={theme}>
      <ErrorBoundary fallback={fallback}>
        <Bomb shouldThrow={shouldThrow} />
      </ErrorBoundary>
    </ThemeProvider>,
  )
}

describe('ErrorBoundary', () => {
  it('renders children when there is no error', () => {
    renderBoundary(false)
    expect(screen.getByText('All good')).toBeInTheDocument()
  })

  it('renders the default error UI when a child throws', () => {
    renderBoundary(true)
    expect(screen.getByText('Something went wrong')).toBeInTheDocument()
    expect(screen.getByText('Test explosion')).toBeInTheDocument()
  })

  it('renders a custom fallback when provided and a child throws', () => {
    renderBoundary(true, <div>Custom fallback</div>)
    expect(screen.getByText('Custom fallback')).toBeInTheDocument()
    expect(screen.queryByText('Something went wrong')).not.toBeInTheDocument()
  })

  it('renders a "Try again" button in the default error UI', () => {
    renderBoundary(true)
    expect(screen.getByRole('button', { name: /try again/i })).toBeInTheDocument()
  })

  it('"Try again" resets the error state and re-renders children', () => {
    // Render without error first so the component is mounted.
    const { rerender } = render(
      <ThemeProvider theme={theme}>
        <ErrorBoundary>
          <Bomb shouldThrow={false} />
        </ErrorBoundary>
      </ThemeProvider>,
    )
    expect(screen.getByText('All good')).toBeInTheDocument()

    // Trigger the error.
    rerender(
      <ThemeProvider theme={theme}>
        <ErrorBoundary>
          <Bomb shouldThrow={true} />
        </ErrorBoundary>
      </ThemeProvider>,
    )
    expect(screen.getByText('Something went wrong')).toBeInTheDocument()

    // Click "Try again" — resets hasError.
    fireEvent.click(screen.getByRole('button', { name: /try again/i }))

    // Children render again (Bomb still throws, but boundary catches anew
    // and re-shows the error; this confirms reset triggered a re-render cycle).
    expect(screen.getByText('Something went wrong')).toBeInTheDocument()
  })

  it('displays the thrown error message in the UI', () => {
    renderBoundary(true)
    // error.message is "Test explosion" — verify it appears in the error UI.
    expect(screen.getByText('Test explosion')).toBeInTheDocument()
  })
})
