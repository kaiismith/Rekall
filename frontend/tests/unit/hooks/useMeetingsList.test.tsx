import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import type { ReactNode } from 'react'

vi.mock('@/services/meetingService', () => ({
  meetingService: {
    listMine: vi.fn().mockResolvedValue({ data: [] }),
  },
}))

import { meetingService } from '@/services/meetingService'
import { useMeetingsList } from '@/hooks/useMeetingsList'

function makeWrapper(initialEntries = ['/']) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return function Wrapper({ children }: { children: ReactNode }) {
    return (
      <MemoryRouter initialEntries={initialEntries}>
        <QueryClientProvider client={queryClient}>
          {children}
        </QueryClientProvider>
      </MemoryRouter>
    )
  }
}

describe('useMeetingsList', () => {
  beforeEach(() => vi.clearAllMocks())

  it('returns empty meetings array initially', async () => {
    const { result } = renderHook(() => useMeetingsList(), { wrapper: makeWrapper() })
    expect(result.current.meetings).toEqual([])
  })

  it('calls meetingService.listMine with default sort', async () => {
    renderHook(() => useMeetingsList(), { wrapper: makeWrapper() })
    await vi.waitFor(() => expect(meetingService.listMine).toHaveBeenCalled())
    expect(meetingService.listMine).toHaveBeenCalledWith({ status: undefined, sort: 'created_at_desc' })
  })

  it('reads status from search params', async () => {
    renderHook(() => useMeetingsList(), {
      wrapper: makeWrapper(['/?status=in_progress']),
    })
    await vi.waitFor(() => expect(meetingService.listMine).toHaveBeenCalled())
    expect(meetingService.listMine).toHaveBeenCalledWith({ status: 'in_progress', sort: 'created_at_desc' })
  })

  it('reads sort from search params', async () => {
    renderHook(() => useMeetingsList(), {
      wrapper: makeWrapper(['/?sort=duration_desc']),
    })
    await vi.waitFor(() => expect(meetingService.listMine).toHaveBeenCalled())
    expect(meetingService.listMine).toHaveBeenCalledWith({ status: undefined, sort: 'duration_desc' })
  })

  it('setStatus() updates the search params', async () => {
    const { result } = renderHook(() => useMeetingsList(), { wrapper: makeWrapper() })
    act(() => result.current.setStatus('complete'))
    expect(result.current.status).toBe('complete')
  })

  it('setStatus(undefined) clears the status param', async () => {
    const { result } = renderHook(() => useMeetingsList(), {
      wrapper: makeWrapper(['/?status=in_progress']),
    })
    act(() => result.current.setStatus(undefined))
    expect(result.current.status).toBeUndefined()
  })

  it('setSort() updates the search params', async () => {
    const { result } = renderHook(() => useMeetingsList(), { wrapper: makeWrapper() })
    act(() => result.current.setSort('duration_asc'))
    expect(result.current.sort).toBe('duration_asc')
  })

  it('setSort() with default sort removes the param', async () => {
    const { result } = renderHook(() => useMeetingsList(), {
      wrapper: makeWrapper(['/?sort=duration_asc']),
    })
    act(() => result.current.setSort('created_at_desc'))
    expect(result.current.sort).toBe('created_at_desc')
  })

  it('activeFilterCount is 0 when no status filter is set', () => {
    const { result } = renderHook(() => useMeetingsList(), { wrapper: makeWrapper() })
    expect(result.current.activeFilterCount).toBe(0)
  })

  it('activeFilterCount is 1 when a status filter is set', () => {
    const { result } = renderHook(() => useMeetingsList(), {
      wrapper: makeWrapper(['/?status=complete']),
    })
    expect(result.current.activeFilterCount).toBe(1)
  })

  it('exposes isLoading and isError from React Query', () => {
    const { result } = renderHook(() => useMeetingsList(), { wrapper: makeWrapper() })
    expect(typeof result.current.isLoading).toBe('boolean')
    expect(typeof result.current.isError).toBe('boolean')
  })
})
