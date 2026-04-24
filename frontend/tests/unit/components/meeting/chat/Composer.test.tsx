import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { Composer } from '@/components/meeting/chat/Composer'

function renderComposer(props: Partial<Parameters<typeof Composer>[0]> = {}) {
  const defaults = { disabled: false, onSend: vi.fn() }
  return render(
    <ThemeProvider theme={theme}>
      <Composer {...defaults} {...props} />
    </ThemeProvider>,
  )
}

function getTextarea(): HTMLTextAreaElement {
  return screen.getByLabelText(/chat message/i) as HTMLTextAreaElement
}

describe('Composer', () => {
  it('sends on Enter (without Shift)', () => {
    const onSend = vi.fn()
    renderComposer({ onSend })
    const ta = getTextarea()
    fireEvent.change(ta, { target: { value: 'hello' } })
    fireEvent.keyDown(ta, { key: 'Enter' })
    expect(onSend).toHaveBeenCalledWith('hello')
  })

  it('inserts newline on Shift+Enter (does not submit)', () => {
    const onSend = vi.fn()
    renderComposer({ onSend })
    const ta = getTextarea()
    fireEvent.change(ta, { target: { value: 'line1' } })
    fireEvent.keyDown(ta, { key: 'Enter', shiftKey: true })
    expect(onSend).not.toHaveBeenCalled()
  })

  it('ignores whitespace-only input', () => {
    const onSend = vi.fn()
    renderComposer({ onSend })
    const ta = getTextarea()
    fireEvent.change(ta, { target: { value: '   \n  ' } })
    fireEvent.keyDown(ta, { key: 'Enter' })
    expect(onSend).not.toHaveBeenCalled()
  })

  it('clears the input after a successful send', () => {
    renderComposer({ onSend: vi.fn() })
    const ta = getTextarea()
    fireEvent.change(ta, { target: { value: 'hello' } })
    fireEvent.keyDown(ta, { key: 'Enter' })
    expect(ta.value).toBe('')
  })

  it('clicking send button submits the value', () => {
    const onSend = vi.fn()
    renderComposer({ onSend })
    fireEvent.change(getTextarea(), { target: { value: 'click send' } })
    fireEvent.click(screen.getByRole('button', { name: 'Send message' }))
    expect(onSend).toHaveBeenCalledWith('click send')
  })

  it('disables the send button when empty', () => {
    renderComposer({ onSend: vi.fn() })
    const btn = screen.getByRole('button', { name: 'Send message' }) as HTMLButtonElement
    expect(btn.disabled).toBe(true)
  })

  it('disables the composer entirely when disabled=true', () => {
    renderComposer({ disabled: true })
    const ta = getTextarea()
    expect(ta.disabled).toBe(true)
  })

  it('shows a character counter', () => {
    renderComposer({ onSend: vi.fn() })
    fireEvent.change(getTextarea(), { target: { value: 'abc' } })
    expect(screen.getByText('3/2000')).toBeInTheDocument()
  })

  it('caps input at MAX_MESSAGE_LENGTH characters', () => {
    renderComposer({ onSend: vi.fn() })
    const ta = getTextarea()
    fireEvent.change(ta, { target: { value: 'a'.repeat(2500) } })
    expect(ta.value.length).toBe(2000)
  })

  it('calls onEscEmpty when Esc is pressed with empty input', () => {
    const onEscEmpty = vi.fn()
    renderComposer({ onSend: vi.fn(), onEscEmpty })
    fireEvent.keyDown(getTextarea(), { key: 'Escape' })
    expect(onEscEmpty).toHaveBeenCalledTimes(1)
  })

  it('does NOT call onEscEmpty when Esc is pressed with non-empty input', () => {
    const onEscEmpty = vi.fn()
    renderComposer({ onSend: vi.fn(), onEscEmpty })
    fireEvent.change(getTextarea(), { target: { value: 'draft' } })
    fireEvent.keyDown(getTextarea(), { key: 'Escape' })
    expect(onEscEmpty).not.toHaveBeenCalled()
  })
})
