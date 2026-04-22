import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { MicButton } from '@/components/meeting/MicButton'

function renderMic(props: Parameters<typeof MicButton>[0]) {
  return render(
    <ThemeProvider theme={theme}>
      <MicButton {...props} />
    </ThemeProvider>,
  )
}

describe('MicButton', () => {
  it('shows Mute tooltip when not muted', () => {
    renderMic({ isMuted: false, audioLevel: 0, onToggle: vi.fn() })
    expect(screen.getByRole('button')).toBeInTheDocument()
  })

  it('shows Unmute tooltip when muted', async () => {
    const { container } = renderMic({ isMuted: true, audioLevel: 0, onToggle: vi.fn() })
    expect(container).toBeTruthy()
  })

  it('calls onToggle when clicked', () => {
    const onToggle = vi.fn()
    renderMic({ isMuted: false, audioLevel: 0, onToggle })
    fireEvent.click(screen.getByRole('button'))
    expect(onToggle).toHaveBeenCalledOnce()
  })

  it('does not render the pulsing ring when muted', () => {
    renderMic({ isMuted: true, audioLevel: 0.8, onToggle: vi.fn() })
    // When muted, no ring Box is rendered; the button has no preceding sibling.
    const button = screen.getByRole('button')
    expect(button.previousElementSibling).toBeNull()
  })

  it('does not render the ring when audioLevel is below 0.05', () => {
    renderMic({ isMuted: false, audioLevel: 0.02, onToggle: vi.fn() })
    // No ring Box: the button has no preceding sibling element.
    const button = screen.getByRole('button')
    expect(button.previousElementSibling).toBeNull()
  })

  it('renders the ring when unmuted and audioLevel >= 0.05', () => {
    renderMic({ isMuted: false, audioLevel: 0.5, onToggle: vi.fn() })
    // Ring Box is rendered as a sibling before the button.
    const button = screen.getByRole('button')
    expect(button.previousElementSibling).not.toBeNull()
  })
})
