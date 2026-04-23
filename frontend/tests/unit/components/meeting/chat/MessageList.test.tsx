import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { MessageList } from '@/components/meeting/chat/MessageList'
import type { ChatMessage, ParticipantDirectoryEntry } from '@/types/meeting'

// JSDOM doesn't implement scrollTop / scrollHeight meaningfully; stub scroll
// behaviour so auto-scroll tests don't throw.
beforeEach(() => {
  Object.defineProperty(HTMLDivElement.prototype, 'scrollTop', {
    configurable: true,
    get() { return (this as unknown as { __scrollTop?: number }).__scrollTop ?? 0 },
    set(v: number) { (this as unknown as { __scrollTop?: number }).__scrollTop = v },
  })
  Object.defineProperty(HTMLDivElement.prototype, 'scrollHeight', {
    configurable: true,
    get() { return 1000 },
  })
  Object.defineProperty(HTMLDivElement.prototype, 'clientHeight', {
    configurable: true,
    get() { return 500 },
  })
})

const directory: Record<string, ParticipantDirectoryEntry> = {
  'u1': { full_name: 'Alice', initials: 'AL' },
  'u2': { full_name: 'Bob', initials: 'BO' },
}

function msg(id: string, userId: string, body: string, sentAt: number, overrides: Partial<ChatMessage> = {}): ChatMessage {
  return { id, userId, body, sentAt, ...overrides }
}

function renderList(props: Partial<Parameters<typeof MessageList>[0]> = {}) {
  const defaults: Parameters<typeof MessageList>[0] = {
    messages: [],
    directory,
    localUserId: 'u1',
    hasMore: false,
    isLoading: false,
    error: null,
    onLoadOlder: vi.fn(),
    onRetry: vi.fn(),
  }
  return render(
    <ThemeProvider theme={theme}>
      <MessageList {...defaults} {...props} />
    </ThemeProvider>,
  )
}

describe('MessageList', () => {
  it('shows empty state when there are no messages', () => {
    renderList()
    expect(screen.getByText(/no messages yet/i)).toBeInTheDocument()
  })

  it('renders messages in chronological order', () => {
    const base = Date.now() - 60_000
    const messages = [
      msg('a', 'u1', 'first', base),
      msg('b', 'u2', 'second', base + 1000),
      msg('c', 'u1', 'third', base + 2000),
    ]
    renderList({ messages })
    const rows = screen.getAllByTestId('chat-message-row')
    expect(rows).toHaveLength(3)
    expect(rows[0].textContent).toContain('first')
    expect(rows[1].textContent).toContain('second')
    expect(rows[2].textContent).toContain('third')
  })

  it('groups consecutive messages from the same sender (second omits avatar/name)', () => {
    const base = Date.now() - 60_000
    const messages = [
      msg('a', 'u1', 'first', base),
      msg('b', 'u1', 'second', base + 30_000), // within 3-min window
    ]
    renderList({ messages })
    const rows = screen.getAllByTestId('chat-message-row')
    // First row has Alice; second should not duplicate the name.
    const aliceCount = screen.getAllByText('Alice').length
    expect(aliceCount).toBe(1)
    expect(rows).toHaveLength(2)
  })

  it('does NOT group when the gap is larger than the window', () => {
    const base = Date.now() - 10 * 60_000
    const messages = [
      msg('a', 'u1', 'first', base),
      msg('b', 'u1', 'second', base + 5 * 60_000), // 5 min later
    ]
    renderList({ messages })
    expect(screen.getAllByText('Alice')).toHaveLength(2)
  })

  it('shows the "Load older" button when hasMore=true', () => {
    renderList({ hasMore: true, messages: [msg('a', 'u1', 'x', Date.now())] })
    expect(screen.getByRole('button', { name: /load older/i })).toBeInTheDocument()
  })

  it('calls onLoadOlder when the button is clicked', () => {
    const onLoadOlder = vi.fn()
    renderList({ hasMore: true, messages: [msg('a', 'u1', 'x', Date.now())], onLoadOlder })
    fireEvent.click(screen.getByRole('button', { name: /load older/i }))
    expect(onLoadOlder).toHaveBeenCalledTimes(1)
  })

  it('shows an error row with Retry button when error prop is set', () => {
    const onRetry = vi.fn()
    renderList({ error: 'network down', onRetry })
    expect(screen.getByText('network down')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /retry/i }))
    expect(onRetry).toHaveBeenCalledTimes(1)
  })

  it('renders linkified URLs in message bodies', () => {
    const { container } = renderList({
      messages: [msg('a', 'u1', 'go https://example.com', Date.now())],
    })
    expect(container.querySelector('a')?.getAttribute('href')).toBe('https://example.com')
  })
})
