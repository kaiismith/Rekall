import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { ThemeProvider } from '@mui/material/styles'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import theme from '@/theme'
import { RecordsPage } from '@/pages/RecordsPage'
import { useOrgsStore } from '@/store/orgsStore'
import { useDeptsStore } from '@/store/deptsStore'

const ORG = '00000000-0000-0000-0000-00000000a001'

vi.mock('@/services/meetingService', () => ({
  meetingService: {
    listMine: vi.fn(),
    create: vi.fn(),
    getByCode: vi.fn(),
    end: vi.fn(),
    listMessages: vi.fn(),
    requestWsTicket: vi.fn(),
    buildAbsoluteWsUrl: vi.fn(),
  },
}))
import * as meetingServiceModule from '@/services/meetingService'

const baseList = {
  success: true as const,
  data: [
    {
      id: 'meet-1',
      code: 'abc-defg-hij',
      title: 'Open Sync',
      type: 'open' as const,
      host_id: 'user-1',
      status: 'ended' as const,
      max_participants: 50,
      transcription_enabled: false,
      join_url: '/meeting/abc-defg-hij',
      created_at: '2026-04-01T10:00:00Z',
      participant_previews: [],
    },
    {
      id: 'meet-2',
      code: 'org-meeting-1',
      title: 'Acme Standup',
      type: 'open' as const,
      host_id: 'user-1',
      status: 'ended' as const,
      max_participants: 50,
      transcription_enabled: false,
      join_url: '/meeting/org-meeting-1',
      created_at: '2026-04-02T10:00:00Z',
      participant_previews: [],
      scope_type: 'organization' as const,
      scope_id: ORG,
    },
  ],
  pagination: { page: 1, per_page: 5, total: 2, total_pages: 1, has_more: false },
}

function renderPage(initialEntries = ['/records']) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <ThemeProvider theme={theme}>
        <MemoryRouter initialEntries={initialEntries}>
          <Routes>
            <Route path="/records" element={<RecordsPage />} />
          </Routes>
        </MemoryRouter>
      </ThemeProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  useOrgsStore.setState({
    orgs: [
      {
        id: ORG,
        name: 'Acme',
        slug: 'acme',
        owner_id: 'u',
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
      },
    ],
    isLoading: false,
    error: null,
  })
  useDeptsStore.setState({ byOrg: {}, isLoading: {}, errors: {} })
  listMineSpy.mockResolvedValue(baseList)
})

// eslint-disable-next-line @typescript-eslint/unbound-method
const listMineSpy = vi.mocked(meetingServiceModule.meetingService.listMine)

describe('RecordsPage scope UI (Task 12.5)', () => {
  it('renders an Open badge for open meetings and an Acme badge for org-scoped ones', async () => {
    renderPage()
    await waitFor(() => expect(screen.getByText('Open Sync')).toBeInTheDocument())
    expect(screen.getByLabelText('Open')).toBeInTheDocument()
    expect(screen.getByLabelText('Acme')).toBeInTheDocument()
  })

  it('renders the scope picker chip in the page header', async () => {
    renderPage()
    await waitFor(() => expect(screen.getByText('Open Sync')).toBeInTheDocument())
    // Default chip text when no scope is selected.
    expect(screen.getByText('All scopes')).toBeInTheDocument()
  })

  it('forwards the URL ?scope=org:<uuid> filter into the listMine call', async () => {
    renderPage([`/records?scope=org:${ORG}`])
    await waitFor(() => {
      expect(listMineSpy).toHaveBeenCalledWith(
        expect.any(Object),
        expect.objectContaining({ type: 'organization', id: ORG }),
      )
    })
  })
})
