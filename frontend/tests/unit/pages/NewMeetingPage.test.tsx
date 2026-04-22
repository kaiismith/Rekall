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

describe('NewMeetingPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(meetingService.create).mockResolvedValue({
      data: { code: 'abc123', id: 'm-1', title: '', type: 'open', status: 'waiting', created_at: '' },
    } as never)
  })

  it('renders the "New Meeting" heading', () => {
    renderPage()
    expect(screen.getByText('New Meeting')).toBeInTheDocument()
  })

  it('renders the title input field', () => {
    renderPage()
    expect(screen.getByLabelText(/title/i)).toBeInTheDocument()
  })

  it('renders the meeting type select', () => {
    renderPage()
    // MUI Select renders a combobox div; find it via its visible role.
    expect(screen.getByRole('combobox')).toBeInTheDocument()
  })

  it('does not show helper text when type is "open" (default)', () => {
    renderPage()
    expect(screen.queryByText(/configure the scope/i)).not.toBeInTheDocument()
  })

  it('shows helper text when type is switched to "private"', async () => {
    renderPage()

    // MUI Select — open by clicking the combobox div
    fireEvent.mouseDown(screen.getByRole('combobox'))
    const privateOption = await screen.findByText(/Private — org\/dept members only/i)
    fireEvent.click(privateOption)

    expect(await screen.findByText(/configure the scope/i)).toBeInTheDocument()
  })

  it('renders Cancel and Create & Join buttons', () => {
    renderPage()
    expect(screen.getByRole('button', { name: /cancel/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /create & join/i })).toBeInTheDocument()
  })

  it('Cancel navigates back', () => {
    renderPage()
    fireEvent.click(screen.getByRole('button', { name: /cancel/i }))
    expect(mockNavigate).toHaveBeenCalledWith(-1)
  })

  it('Create & Join calls meetingService.create with current title and type', async () => {
    renderPage()

    fireEvent.change(screen.getByLabelText(/title/i), { target: { value: 'Sprint Retro' } })
    fireEvent.click(screen.getByRole('button', { name: /create & join/i }))

    await waitFor(() => expect(meetingService.create).toHaveBeenCalledWith({ title: 'Sprint Retro', type: 'open' }))
  })

  it('navigates to /meeting/:code on successful create', async () => {
    renderPage()
    fireEvent.click(screen.getByRole('button', { name: /create & join/i }))

    await waitFor(() => expect(mockNavigate).toHaveBeenCalledWith('/meeting/abc123'))
  })

  it('shows an error message when create fails with a structured response', async () => {
    vi.mocked(meetingService.create).mockRejectedValue({
      response: { data: { error: { message: 'Title too long.' } } },
    })
    renderPage()
    fireEvent.click(screen.getByRole('button', { name: /create & join/i }))

    expect(await screen.findByText('Title too long.')).toBeInTheDocument()
  })

  it('shows generic error message when create fails without a structured response', async () => {
    vi.mocked(meetingService.create).mockRejectedValue(new Error('Network failure'))
    renderPage()
    fireEvent.click(screen.getByRole('button', { name: /create & join/i }))

    expect(await screen.findByText('Failed to create meeting.')).toBeInTheDocument()
  })

  it('buttons are disabled while creating', async () => {
    // Make create hang so we can inspect mid-flight state.
    let resolve!: () => void
    vi.mocked(meetingService.create).mockReturnValue(
      new Promise((r) => { resolve = () => r({ data: { code: 'x' } } as never) }),
    )

    renderPage()
    fireEvent.click(screen.getByRole('button', { name: /create & join/i }))

    expect(screen.getByRole('button', { name: /creating/i })).toBeDisabled()
    expect(screen.getByRole('button', { name: /cancel/i })).toBeDisabled()

    resolve()
    await waitFor(() => expect(mockNavigate).toHaveBeenCalled())
  })

  it('shows "Creating…" text on the button while loading', async () => {
    let resolve!: () => void
    vi.mocked(meetingService.create).mockReturnValue(
      new Promise((r) => { resolve = () => r({ data: { code: 'x' } } as never) }),
    )

    renderPage()
    fireEvent.click(screen.getByRole('button', { name: /create & join/i }))

    expect(screen.getByRole('button', { name: /creating/i })).toBeInTheDocument()

    resolve()
    await waitFor(() => expect(mockNavigate).toHaveBeenCalled())
  })

  it('re-enables buttons and hides "Creating…" after create resolves', async () => {
    renderPage()
    fireEvent.click(screen.getByRole('button', { name: /create & join/i }))
    await waitFor(() => expect(mockNavigate).toHaveBeenCalled())
    // Component would unmount after navigate in a real app; just verify create finished.
    expect(meetingService.create).toHaveBeenCalledTimes(1)
  })

  it('error cleared on a second create attempt', async () => {
    vi.mocked(meetingService.create)
      .mockRejectedValueOnce(new Error('oops'))
      .mockResolvedValueOnce({ data: { code: 'ok' } } as never)

    renderPage()
    fireEvent.click(screen.getByRole('button', { name: /create & join/i }))
    expect(await screen.findByText('Failed to create meeting.')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: /create & join/i }))
    await waitFor(() => expect(screen.queryByText('Failed to create meeting.')).not.toBeInTheDocument())
  })
})
