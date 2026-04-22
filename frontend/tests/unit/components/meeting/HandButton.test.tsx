import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { HandButton } from '@/components/meeting/HandButton'

function renderHand(props: Parameters<typeof HandButton>[0]) {
  return render(
    <ThemeProvider theme={theme}>
      <HandButton {...props} />
    </ThemeProvider>,
  )
}

describe('HandButton', () => {
  it('renders a button', () => {
    renderHand({ isHandRaised: false, onToggle: vi.fn() })
    expect(screen.getByRole('button')).toBeInTheDocument()
  })

  it('calls onToggle when clicked', () => {
    const onToggle = vi.fn()
    renderHand({ isHandRaised: false, onToggle })
    fireEvent.click(screen.getByRole('button'))
    expect(onToggle).toHaveBeenCalledOnce()
  })

  it('calls onToggle when hand is raised and clicked', () => {
    const onToggle = vi.fn()
    renderHand({ isHandRaised: true, onToggle })
    fireEvent.click(screen.getByRole('button'))
    expect(onToggle).toHaveBeenCalledOnce()
  })
})
