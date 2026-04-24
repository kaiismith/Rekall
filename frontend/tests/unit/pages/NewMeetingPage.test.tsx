import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import { MemoryRouter } from 'react-router-dom'
import theme from '@/theme'
import { NewMeetingPage } from '@/pages/NewMeetingPage'

// ── mocks ────────────────────────────────────────────────────────────────────

const mockNavigate = vi.fn()
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom')
  return { ...actual, useNavigate: () => mockNavigate }
})

vi.mock('@/services/meetingService', () => ({
  meetingService: {
    create: vi.fn(),
  },
}))

import { meetingService } from '@/services/meetingService'

// ── helpers ───────────────────────────────────────────────────────────────────

function renderPage() {
  return render(
    <ThemeProvider theme={theme}>
      <MemoryRouter>
        <NewMeetingPage />
      </MemoryRouter>
    </ThemeProvider>,
  )
}

// ── tests ─────────────────────────────────────────────────────────────────────

describe('NewMeetingPage — Rekall layout', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(meetingService.create).mockResolvedValue({
      data: { code: 'abc123', id: 'm-1', title: '', type: 'open', status: 'waiting', created_at: '' },
    } as never)
  })

  it('renders the Rekall Meeting hero', () => {
    renderPage()
    expect(screen.getByRole('heading', { name: /rekall meeting/i })).toBeInTheDocument()
    expect(screen.getByText(/create a new meeting or join with a code\./i)).toBeInTheDocument()
  })

  it('renders the Create meeting gradient button', () => {
    renderPage()
    expect(screen.getByRole('button', { name: /create meeting/i })).toBeInTheDocument()
  })

  it('renders the transcript language select', () => {
    renderPage()
    expect(screen.getByText(/transcript language/i)).toBeInTheDocument()
    // MUI Select exposes a combobox role
    expect(screen.getByRole('combobox')).toBeInTheDocument()
  })

  it('renders the Join a meeting section', () => {
    renderPage()
    expect(screen.getByText(/join a meeting/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/meeting code/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^join$/i })).toBeInTheDocument()
  })

  it('Create meeting calls meetingService.create with defaults and navigates', async () => {
    renderPage()
    fireEvent.click(screen.getByRole('button', { name: /create meeting/i }))
    await waitFor(() =>
      expect(meetingService.create).toHaveBeenCalledWith({ title: '', type: 'open' }),
    )
    await waitFor(() => expect(mockNavigate).toHaveBeenCalledWith('/meeting/abc123'))
  })

  it('surfaces a structured error message on create failure', async () => {
    vi.mocked(meetingService.create).mockRejectedValue({
      response: { data: { error: { message: 'Rate-limited.' } } },
    })
    renderPage()
    fireEvent.click(screen.getByRole('button', { name: /create meeting/i }))
    expect(await screen.findByText('Rate-limited.')).toBeInTheDocument()
  })

  it('falls back to generic error message when the response is unstructured', async () => {
    vi.mocked(meetingService.create).mockRejectedValue(new Error('Network failure'))
    renderPage()
    fireEvent.click(screen.getByRole('button', { name: /create meeting/i }))
    // The page now surfaces errors in a modal Dialog with a fallback copy.
    expect(
      await screen.findByText(/Something went wrong\. Please try again\./i),
    ).toBeInTheDocument()
  })

  it('disables the Create button while a request is in flight', async () => {
    let resolve!: () => void
    vi.mocked(meetingService.create).mockReturnValue(
      new Promise((r) => { resolve = () => r({ data: { code: 'x' } } as never) }),
    )
    renderPage()
    fireEvent.click(screen.getByRole('button', { name: /create meeting/i }))
    expect(screen.getByRole('button', { name: /creating/i })).toBeDisabled()
    resolve()
    await waitFor(() => expect(mockNavigate).toHaveBeenCalled())
  })

  // ── Join section ────────────────────────────────────────────────────────────

  it('Join button is disabled when the code is shorter than the minimum', () => {
    renderPage()
    const input = screen.getByLabelText(/meeting code/i)
    fireEvent.change(input, { target: { value: 'abc' } })
    expect(screen.getByRole('button', { name: /^join$/i })).toBeDisabled()
  })

  it('Join button is enabled when the code reaches the minimum length', () => {
    renderPage()
    const input = screen.getByLabelText(/meeting code/i)
    fireEvent.change(input, { target: { value: 'abcdef' } })
    expect(screen.getByRole('button', { name: /^join$/i })).toBeEnabled()
  })

  it('clicking Join navigates to /meeting/<code>', () => {
    renderPage()
    const input = screen.getByLabelText(/meeting code/i)
    fireEvent.change(input, { target: { value: 'abcdef' } })
    fireEvent.click(screen.getByRole('button', { name: /^join$/i }))
    expect(mockNavigate).toHaveBeenCalledWith('/meeting/abcdef')
  })

  it('pressing Enter inside the code input navigates when valid', () => {
    renderPage()
    const input = screen.getByLabelText(/meeting code/i)
    fireEvent.change(input, { target: { value: 'xyz123' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(mockNavigate).toHaveBeenCalledWith('/meeting/xyz123')
  })

  it('lowercases and trims whitespace from the code input', () => {
    renderPage()
    const input = screen.getByLabelText(/meeting code/i) as HTMLInputElement
    fireEvent.change(input, { target: { value: '  ABCDEF  ' } })
    expect(input.value).toBe('abcdef')
  })

  // ── Keyboard shortcut ───────────────────────────────────────────────────────

  it('Ctrl+Shift+C triggers a create', async () => {
    renderPage()
    fireEvent.keyDown(document, { key: 'c', ctrlKey: true, shiftKey: true })
    await waitFor(() => expect(meetingService.create).toHaveBeenCalled())
  })

  it('the shortcut is ignored while the user is typing in an input', () => {
    renderPage()
    const input = screen.getByLabelText(/meeting code/i)
    input.focus()
    fireEvent.keyDown(input, { key: 'c', ctrlKey: true, shiftKey: true })
    expect(meetingService.create).not.toHaveBeenCalled()
  })
})
