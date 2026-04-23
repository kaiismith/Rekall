import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { ChatButton } from '@/components/meeting/ChatButton'

function renderButton(props: Parameters<typeof ChatButton>[0]) {
  return render(
    <ThemeProvider theme={theme}>
      <ChatButton {...props} />
    </ThemeProvider>,
  )
}

describe('ChatButton', () => {
  it('uses plain "Open chat" aria-label when there are no unread messages', () => {
    renderButton({ unreadCount: 0, isOpen: false, onToggle: vi.fn() })
    expect(screen.getByRole('button', { name: 'Open chat' })).toBeInTheDocument()
  })

  it('announces the unread count via aria-label when > 0', () => {
    renderButton({ unreadCount: 5, isOpen: false, onToggle: vi.fn() })
    expect(
      screen.getByRole('button', { name: /open chat, 5 unread messages/i }),
    ).toBeInTheDocument()
  })

  it('uses singular "message" when the count is exactly 1', () => {
    renderButton({ unreadCount: 1, isOpen: false, onToggle: vi.fn() })
    expect(
      screen.getByRole('button', { name: /open chat, 1 unread message$/i }),
    ).toBeInTheDocument()
  })

  it('uses "Close chat" label when already open', () => {
    renderButton({ unreadCount: 3, isOpen: true, onToggle: vi.fn() })
    expect(screen.getByRole('button', { name: 'Close chat' })).toBeInTheDocument()
  })

  it('caps display at 99+ for very large counts', () => {
    renderButton({ unreadCount: 150, isOpen: false, onToggle: vi.fn() })
    // MUI Badge uses max=99 → renders "99+" text content.
    expect(screen.getByText(/99\+/)).toBeInTheDocument()
  })

  it('calls onToggle when clicked', () => {
    const onToggle = vi.fn()
    renderButton({ unreadCount: 0, isOpen: false, onToggle })
    fireEvent.click(screen.getByRole('button', { name: 'Open chat' }))
    expect(onToggle).toHaveBeenCalledTimes(1)
  })
})
