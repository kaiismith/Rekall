import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { ThemeProvider } from '@mui/material/styles'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import theme from '@/theme'
import { CallsPage } from '@/pages/CallsPage'
import * as callServiceModule from '@/services/callService'
import { ApiError } from '@/services/api'

vi.mock('@/services/callService', () => ({
  callService: {
    list: vi.fn(),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}))

// CallsPage uses usePagination, formatDateTime, formatDuration — these depend on
// real implementations but we test the page-level behaviour only.

const mockCallsPage = {
  success: true as const,
  data: [
    {
      id: 'call-1',
      user_id: 'user-1',
      title: 'Q1 Sales Discovery',
      status: 'done' as const,
      duration_sec: 1800,
      metadata: {},
      created_at: '2024-01-15T10:00:00Z',
      updated_at: '2024-01-15T10:30:00Z',
    },
    {
      id: 'call-2',
      user_id: 'user-1',
      title: 'Onboarding Call',
      status: 'pending' as const,
      duration_sec: 0,
      metadata: {},
      created_at: '2024-01-16T14:00:00Z',
      updated_at: '2024-01-16T14:00:00Z',
    },
  ],
  meta: { page: 1, per_page: 20, total: 2 },
}

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <ThemeProvider theme={theme}>
        <MemoryRouter>
          <CallsPage />
        </MemoryRouter>
      </ThemeProvider>
    </QueryClientProvider>,
  )
}

describe('CallsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  // ── Loading state ────────────────────────────────────────────────────────────

  it('shows skeleton rows while data is loading', () => {
    vi.mocked(callServiceModule.callService.list).mockReturnValue(new Promise(() => {}))

    renderPage()

    // Skeleton rows are rendered as table rows
    const rows = screen.getAllByRole('row')
    // Header row + 5 skeleton rows
    expect(rows.length).toBeGreaterThan(1)
  })

  // ── Empty state ──────────────────────────────────────────────────────────────

  it('shows empty state when no calls exist', async () => {
    vi.mocked(callServiceModule.callService.list).mockResolvedValue({ success: true, data: [], meta: { page: 1, per_page: 20, total: 0 } })

    renderPage()

    await waitFor(() => {
      expect(screen.getByText(/no calls yet/i)).toBeInTheDocument()
    })
  })

  // ── Calls table ──────────────────────────────────────────────────────────────

  it('renders page title', async () => {
    vi.mocked(callServiceModule.callService.list).mockResolvedValue(mockCallsPage)

    renderPage()

    await waitFor(() => {
      expect(screen.getByText('Calls')).toBeInTheDocument()
    })
  })

  it('renders all call titles from the API', async () => {
    vi.mocked(callServiceModule.callService.list).mockResolvedValue(mockCallsPage)

    renderPage()

    await waitFor(() => {
      expect(screen.getByText('Q1 Sales Discovery')).toBeInTheDocument()
      expect(screen.getByText('Onboarding Call')).toBeInTheDocument()
    })
  })

  it('renders status chips for each call', async () => {
    vi.mocked(callServiceModule.callService.list).mockResolvedValue(mockCallsPage)

    renderPage()

    await waitFor(() => {
      // "done" and "pending" status chips should appear
      expect(screen.getByText(/done/i)).toBeInTheDocument()
      expect(screen.getByText(/pending/i)).toBeInTheDocument()
    })
  })

  it('renders correct table column headers', async () => {
    vi.mocked(callServiceModule.callService.list).mockResolvedValue(mockCallsPage)

    renderPage()

    await waitFor(() => {
      expect(screen.getByText('Title')).toBeInTheDocument()
      expect(screen.getByText('Status')).toBeInTheDocument()
      expect(screen.getByText('Duration')).toBeInTheDocument()
      expect(screen.getByRole('columnheader', { name: /created/i })).toBeInTheDocument()
    })
  })

  // ── Pagination ───────────────────────────────────────────────────────────────

  it('renders pagination component', async () => {
    vi.mocked(callServiceModule.callService.list).mockResolvedValue(mockCallsPage)

    renderPage()

    await waitFor(() => {
      // TablePagination renders rows-per-page selector
      expect(screen.getByRole('combobox')).toBeInTheDocument()
    })
  })

  it('passes correct pagination params to the service', async () => {
    vi.mocked(callServiceModule.callService.list).mockResolvedValue(mockCallsPage)

    renderPage()

    await waitFor(() => {
      expect(callServiceModule.callService.list).toHaveBeenCalledWith(
        expect.objectContaining({ page: 1 }),
      )
    })
  })

  // ── Error state ──────────────────────────────────────────────────────────────

  it('shows error message in table when API fails', async () => {
    vi.mocked(callServiceModule.callService.list).mockRejectedValue(
      new ApiError('INTERNAL_ERROR', 'service unavailable', 500),
    )

    renderPage()

    await waitFor(() => {
      expect(screen.getByText('service unavailable')).toBeInTheDocument()
    })
  })

  it('shows error message from Error instances', async () => {
    vi.mocked(callServiceModule.callService.list).mockRejectedValue(
      new Error('Network timeout'),
    )

    renderPage()

    await waitFor(() => {
      // Component renders error.message for Error instances
      expect(screen.getByText('Network timeout')).toBeInTheDocument()
    })
  })
})
