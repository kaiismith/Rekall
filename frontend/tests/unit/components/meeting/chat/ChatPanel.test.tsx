import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { ChatPanel } from '@/components/meeting/chat/ChatPanel'
import type { ChatMessage } from '@/types/meeting'

// Stub scroll props as MessageList reads them in its effects.
beforeEach(() => {
  Object.defineProperty(HTMLDivElement.prototype, 'scrollTop', {
    configurable: true,
    get() { return 0 },
    set() { /* no-op */ },
  })
  Object.defineProperty(HTMLDivElement.prototype, 'scrollHeight', { configurable: true, get() { return 100 } })
  Object.defineProperty(HTMLDivElement.prototype, 'clientHeight', { configurable: true, get() { return 100 } })
})

function renderPanel(props: Partial<Parameters<typeof ChatPanel>[0]> = {}) {
  const defaults: Parameters<typeof ChatPanel>[0] = {
    isOpen: true,
    onClose: vi.fn(),
    messages: [],
    directory: {},
    localUserId: 'u1',
    hasMore: false,
    isLoading: false,
    historyError: null,
    sendError: null,
    onLoadOlder: vi.fn(),
    onRetry: vi.fn(),
    onSend: vi.fn(),
    onDismissSendError: vi.fn(),
    composerDisabled: false,
    flashKey: 0,
  }
  return render(
    <ThemeProvider theme={theme}>
      <ChatPanel {...defaults} {...props} />
    </ThemeProvider>,
  )
}

describe('ChatPanel', () => {
  it('is visible when isOpen=true', () => {
    renderPanel({ isOpen: true })
    const panel = screen.getByTestId('chat-panel')
    expect(panel).toBeInTheDocument()
    expect(panel.getAttribute('aria-hidden')).toBe('false')
  })

  it('stays mounted but hidden when isOpen=false', () => {
    renderPanel({ isOpen: false })
    const panel = screen.getByTestId('chat-panel')
    // Mounted (in the DOM) but aria-hidden signals to consumers it is closed.
    expect(panel).toBeInTheDocument()
    expect(panel.getAttribute('aria-hidden')).toBe('true')
    // Composer textarea is still in the DOM so live events can keep state up
    // to date without re-fetching history on re-open.
    expect(screen.getByLabelText(/chat message/i)).toBeInTheDocument()
  })

  it('close button triggers onClose', () => {
    const onClose = vi.fn()
    renderPanel({ onClose })
    fireEvent.click(screen.getByLabelText('Close chat'))
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('sends messages via the Composer', () => {
    const onSend = vi.fn()
    renderPanel({ onSend })
    const ta = screen.getByLabelText(/chat message/i)
    fireEvent.change(ta, { target: { value: 'hello panel' } })
    fireEvent.keyDown(ta, { key: 'Enter' })
    expect(onSend).toHaveBeenCalledWith('hello panel')
  })

  it('dismisses send error when user sends again', () => {
    const onSend = vi.fn()
    const onDismissSendError = vi.fn()
    renderPanel({ onSend, onDismissSendError, sendError: 'Not connected' })
    expect(screen.getByText('Not connected')).toBeInTheDocument()
    const ta = screen.getByLabelText(/chat message/i)
    fireEvent.change(ta, { target: { value: 'retry' } })
    fireEvent.keyDown(ta, { key: 'Enter' })
    expect(onDismissSendError).toHaveBeenCalled()
    expect(onSend).toHaveBeenCalledWith('retry')
  })

  it('renders an existing message', () => {
    const messages: ChatMessage[] = [
      { id: 'a', userId: 'u1', body: 'hi there', sentAt: Date.now() - 1000 },
    ]
    renderPanel({
      messages,
      directory: { u1: { full_name: 'Alice', initials: 'AL' } },
    })
    expect(screen.getByText('hi there')).toBeInTheDocument()
    expect(screen.getByText('Alice')).toBeInTheDocument()
  })

  it('disables the composer when composerDisabled=true', () => {
    renderPanel({ composerDisabled: true })
    const ta = screen.getByLabelText(/chat message/i) as HTMLTextAreaElement
    expect(ta.disabled).toBe(true)
  })
})
