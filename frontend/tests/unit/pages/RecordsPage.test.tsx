import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import type * as ReactRouterDom from 'react-router-dom'
import { MemoryRouter } from 'react-router-dom'
import { ThemeProvider } from '@mui/material/styles'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import theme from '@/theme'
import { RecordsPage } from '@/pages/RecordsPage'
import * as meetingServiceModule from '@/services/meetingService'
import type { Meeting } from '@/types/meeting'

vi.mock('@/services/meetingService', () => ({
  meetingService: {
    listMine: vi.fn(),
    create: vi.fn(),
    getByCode: vi.fn(),
    end: vi.fn(),
  },
}))

const mockNavigate = vi.fn()
vi.mock('react-router-dom', async (importOriginal) => {
  const actual = await importOriginal<typeof ReactRouterDom>()
  return { ...actual, useNavigate: () => mockNavigate }
})

function meeting(overrides: Partial<Meeting> = {}): Meeting {
  return {
    id: 'meet-1',
    code: 'abc-defg-hij',
    title: 'Weekly Sync',
    type: 'open',
    host_id: 'user-1',
    status: 'ended',
    max_participants: 50,
    transcription_enabled: false,
    join_url: 'http://localhost/meeting/abc-defg-hij',
    created_at: '2025-03-01T10:00:00Z',
    participant_previews: [],
    ...overrides,
  }
}

function renderPage(initialUrl = '/records') {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <ThemeProvider theme={theme}>
        <MemoryRouter initialEntries={[initialUrl]}>
          <RecordsPage />
        </MemoryRouter>
      </ThemeProvider>
    </QueryClientProvider>,
  )
}

// eslint-disable-next-line @typescript-eslint/unbound-method
const listMineSpy = vi.mocked(meetingServiceModule.meetingService.listMine)

describe('RecordsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders the page heading', () => {
    listMineSpy.mockResolvedValue({ success: true, data: [] })
    renderPage()
    expect(screen.getByText('Your Records')).toBeInTheDocument()
  })

  it('shows empty state when no records exist', async () => {
    listMineSpy.mockResolvedValue({ success: true, data: [] })
    renderPage()
    await waitFor(() => {
      expect(screen.getByText(/no records yet/i)).toBeInTheDocument()
    })
  })

  it('shows "Start a Meeting" CTA in empty state without filter', async () => {
    listMineSpy.mockResolvedValue({ success: true, data: [] })
    renderPage()
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /start a meeting/i })).toBeInTheDocument()
    })
  })

  it('shows "No records match this filter" when filter is active and list is empty', async () => {
    listMineSpy.mockResolvedValue({ success: true, data: [] })
    renderPage('/records?status=complete')
    await waitFor(() => {
      expect(screen.getByText(/no records match this filter/i)).toBeInTheDocument()
    })
  })

  it('renders meeting cards when data is returned', async () => {
    listMineSpy.mockResolvedValue({
      success: true,
      data: [meeting({ title: 'Team Standup' }), meeting({ id: 'meet-2', title: 'Design Review' })],
    })
    renderPage()
    await waitFor(() => {
      expect(screen.getByText('Team Standup')).toBeInTheDocument()
      expect(screen.getByText('Design Review')).toBeInTheDocument()
    })
  })

  it('shows filter badge with count 1 when status param is set', async () => {
    listMineSpy.mockResolvedValue({ success: true, data: [] })
    renderPage('/records?status=complete')
    await waitFor(() => {
      expect(screen.getByText('1')).toBeInTheDocument()
    })
  })

  it('passes status and sort URL params to the service', async () => {
    listMineSpy.mockResolvedValue({ success: true, data: [] })
    renderPage('/records?status=complete&sort=duration_desc')
    await waitFor(() => {
      expect(listMineSpy).toHaveBeenCalledWith(
        expect.objectContaining({ status: 'complete', sort: 'duration_desc' }),
        null,
      )
    })
  })

  it('clicking a meeting card navigates to /records/:code (NOT /meeting/:code)', async () => {
    listMineSpy.mockResolvedValue({
      success: true,
      data: [meeting({ code: 'rec-abc', title: 'Team Standup' })],
    })
    renderPage()
    await waitFor(() => {
      expect(screen.getByText('Team Standup')).toBeInTheDocument()
    })
    fireEvent.click(screen.getByText('Team Standup'))
    expect(mockNavigate).toHaveBeenCalledWith('/records/rec-abc')
  })
})
