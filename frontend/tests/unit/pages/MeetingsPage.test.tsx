import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { ThemeProvider } from '@mui/material/styles'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import theme from '@/theme'
import { MeetingsPage } from '@/pages/MeetingsPage'
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
  const actual = await importOriginal<typeof import('react-router-dom')>()
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
    join_url: 'http://localhost/meeting/abc-defg-hij',
    created_at: '2025-03-01T10:00:00Z',
    participant_previews: [],
    ...overrides,
  }
}

function renderPage(initialUrl = '/meetings') {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <ThemeProvider theme={theme}>
        <MemoryRouter initialEntries={[initialUrl]}>
          <MeetingsPage />
        </MemoryRouter>
      </ThemeProvider>
    </QueryClientProvider>,
  )
}

describe('MeetingsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders the page heading', async () => {
    vi.mocked(meetingServiceModule.meetingService.listMine).mockResolvedValue({
      success: true,
      data: [],
    })
    renderPage()
    expect(screen.getByText('Your Meetings')).toBeInTheDocument()
  })

  it('shows empty state when no meetings exist', async () => {
    vi.mocked(meetingServiceModule.meetingService.listMine).mockResolvedValue({
      success: true,
      data: [],
    })
    renderPage()
    await waitFor(() => {
      expect(screen.getByText(/no meetings yet/i)).toBeInTheDocument()
    })
  })

  it('shows "Start a Meeting" CTA in empty state without filter', async () => {
    vi.mocked(meetingServiceModule.meetingService.listMine).mockResolvedValue({
      success: true,
      data: [],
    })
    renderPage()
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /start a meeting/i })).toBeInTheDocument()
    })
  })

  it('shows "No meetings match this filter" when filter is active and list is empty', async () => {
    vi.mocked(meetingServiceModule.meetingService.listMine).mockResolvedValue({
      success: true,
      data: [],
    })
    renderPage('/meetings?status=complete')
    await waitFor(() => {
      expect(screen.getByText(/no meetings match this filter/i)).toBeInTheDocument()
    })
  })

  it('renders meeting cards when data is returned', async () => {
    vi.mocked(meetingServiceModule.meetingService.listMine).mockResolvedValue({
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
    vi.mocked(meetingServiceModule.meetingService.listMine).mockResolvedValue({
      success: true,
      data: [],
    })
    renderPage('/meetings?status=complete')
    await waitFor(() => {
      // MUI Badge renders badgeContent as a span
      expect(screen.getByText('1')).toBeInTheDocument()
    })
  })

  it('passes status and sort URL params to the service', async () => {
    vi.mocked(meetingServiceModule.meetingService.listMine).mockResolvedValue({
      success: true,
      data: [],
    })
    renderPage('/meetings?status=complete&sort=duration_desc')
    await waitFor(() => {
      expect(meetingServiceModule.meetingService.listMine).toHaveBeenCalledWith(
        expect.objectContaining({ status: 'complete', sort: 'duration_desc' }),
      )
    })
  })
})
