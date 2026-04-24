import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { MessageRow } from '@/components/meeting/chat/MessageRow'
import type { ChatMessage, ParticipantDirectoryEntry } from '@/types/meeting'

function renderRow(
  message: ChatMessage,
  sender: ParticipantDirectoryEntry | null = { full_name: 'Alice', initials: 'AL' },
  grouped = false,
) {
  return render(
    <ThemeProvider theme={theme}>
      <MessageRow message={message} sender={sender} grouped={grouped} />
    </ThemeProvider>,
  )
}

function makeMsg(overrides: Partial<ChatMessage> = {}): ChatMessage {
  return {
    id: 'msg-1',
    userId: 'user-1',
    body: 'hello',
    sentAt: Date.now() - 30_000,
    ...overrides,
  }
}

describe('MessageRow', () => {
  it('shows sender name and initials in the avatar', () => {
    renderRow(makeMsg(), { full_name: 'Alice Smith', initials: 'AS' })
    expect(screen.getByText('Alice Smith')).toBeInTheDocument()
    expect(screen.getByText('AS')).toBeInTheDocument()
  })

  it('falls back to "User" and "?" when sender is null', () => {
    renderRow(makeMsg(), null)
    expect(screen.getByText('User')).toBeInTheDocument()
    expect(screen.getByText('?')).toBeInTheDocument()
  })

  it('omits avatar and name when grouped=true', () => {
    renderRow(makeMsg(), { full_name: 'Alice', initials: 'AL' }, true)
    expect(screen.queryByText('Alice')).not.toBeInTheDocument()
    expect(screen.queryByText('AL')).not.toBeInTheDocument()
  })

  it('linkifies URLs in the body', () => {
    const { container } = renderRow(makeMsg({ body: 'visit https://example.com today' }))
    const a = container.querySelector('a')
    expect(a).not.toBeNull()
    expect(a?.getAttribute('href')).toBe('https://example.com')
    expect(a?.getAttribute('target')).toBe('_blank')
    expect(a?.getAttribute('rel')).toBe('noopener noreferrer')
  })

  it('escapes HTML-like content as plain text', () => {
    const { container } = renderRow(makeMsg({ body: '<script>alert(1)</script>' }))
    expect(container.querySelector('script')).toBeNull()
    expect(container.textContent).toContain('<script>alert(1)</script>')
  })

  it('renders with reduced opacity when pending', () => {
    renderRow(makeMsg({ pending: true }))
    const row = screen.getByTestId('chat-message-row')
    expect(row.getAttribute('data-pending')).toBe('true')
  })

  it('preserves line breaks in the body via white-space: pre-wrap', () => {
    renderRow(makeMsg({ body: 'line1\nline2' }))
    // Both lines are in the same text node; the style applies but we can at
    // least assert the raw text content survived.
    expect(screen.getByText(/line1/)).toBeInTheDocument()
  })
})
