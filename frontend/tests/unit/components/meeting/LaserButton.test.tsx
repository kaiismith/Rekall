import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { LaserButton } from '@/components/meeting/LaserButton'

function renderLaser(props: Parameters<typeof LaserButton>[0]) {
  return render(
    <ThemeProvider theme={theme}>
      <LaserButton {...props} />
    </ThemeProvider>,
  )
}

describe('LaserButton', () => {
  it('renders a button', () => {
    renderLaser({ isActive: false, onToggle: vi.fn() })
    expect(screen.getByRole('button')).toBeInTheDocument()
  })

  it('calls onToggle when clicked', () => {
    const onToggle = vi.fn()
    renderLaser({ isActive: false, onToggle })
    fireEvent.click(screen.getByRole('button'))
    expect(onToggle).toHaveBeenCalledOnce()
  })

  it('calls onToggle when active and clicked', () => {
    const onToggle = vi.fn()
    renderLaser({ isActive: true, onToggle })
    fireEvent.click(screen.getByRole('button'))
    expect(onToggle).toHaveBeenCalledOnce()
  })
})
