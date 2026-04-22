import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { EmojiButton } from '@/components/meeting/EmojiButton'
import { EMOJI_LIST } from '@/config/meetingControls'

function renderEmoji(props: Parameters<typeof EmojiButton>[0]) {
  return render(
    <ThemeProvider theme={theme}>
      <EmojiButton {...props} />
    </ThemeProvider>,
  )
}

describe('EmojiButton', () => {
  it('renders a trigger button', () => {
    renderEmoji({ onSend: vi.fn() })
    expect(screen.getByRole('button')).toBeInTheDocument()
  })

  it('opens emoji picker on click', () => {
    renderEmoji({ onSend: vi.fn() })
    fireEvent.click(screen.getByRole('button'))
    // All emojis in EMOJI_LIST should now be visible.
    EMOJI_LIST.forEach((emoji) => {
      expect(screen.getByText(emoji)).toBeInTheDocument()
    })
  })

  it('calls onSend with the selected emoji', () => {
    const onSend = vi.fn()
    renderEmoji({ onSend })
    fireEvent.click(screen.getByRole('button'))
    fireEvent.click(screen.getByText('👍'))
    expect(onSend).toHaveBeenCalledWith('👍')
  })

  it('closes picker after emoji selection', async () => {
    renderEmoji({ onSend: vi.fn() })
    fireEvent.click(screen.getByRole('button'))
    expect(screen.getByText('👍')).toBeInTheDocument()
    fireEvent.click(screen.getByText('👍'))
    // MUI Popover may keep DOM alive during exit transition — wait for unmount.
    await waitFor(() => {
      expect(screen.queryByText('👎')).not.toBeInTheDocument()
    })
  })
})
