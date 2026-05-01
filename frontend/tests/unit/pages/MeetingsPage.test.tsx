import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import type * as ReactRouterDom from 'react-router-dom'
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

// eslint-disable-next-line @typescript-eslint/unbound-method
const listMineSpy = vi.mocked(meetingServiceModule.meetingService.listMine)

describe('MeetingsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders the page heading', () => {
    listMineSpy.mockResolvedValue({
      success: true,
      data: [],
      pagination: { page: 1, per_page: 5, total: 0, total_pages: 0, has_more: false },
    })
    renderPage()
    expect(screen.getByText('Your Meetings')).toBeInTheDocument()
  })

  it('shows empty state when no meetings exist', async () => {
    listMineSpy.mockResolvedValue({
      success: true,
      data: [],
      pagination: { page: 1, per_page: 5, total: 0, total_pages: 0, has_more: false },
    })
    renderPage()
    await waitFor(() => {
      expect(screen.getByText(/no meetings yet/i)).toBeInTheDocument()
    })
  })

  it('renders meeting cards when data is returned', async () => {
    listMineSpy.mockResolvedValue({
      success: true,
      data: [meeting({ title: 'Team Standup' })],
      pagination: { page: 1, per_page: 5, total: 1, total_pages: 1, has_more: false },
    })
    renderPage()
    await waitFor(() => {
      expect(screen.getByText('Team Standup')).toBeInTheDocument()
    })
  })

  it('clicking a meeting card navigates to /meeting/:code (live room, NOT records detail)', async () => {
    listMineSpy.mockResolvedValue({
      success: true,
      data: [meeting({ code: 'live-abc', title: 'Live Standup' })],
      pagination: { page: 1, per_page: 5, total: 1, total_pages: 1, has_more: false },
    })
    renderPage()
    await waitFor(() => {
      expect(screen.getByText('Live Standup')).toBeInTheDocument()
    })
    fireEvent.click(screen.getByText('Live Standup'))
    expect(mockNavigate).toHaveBeenCalledWith('/meeting/live-abc')
  })
})
