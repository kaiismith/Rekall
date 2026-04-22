import { describe, it, expect, vi, beforeEach } from 'vitest'
import { callService } from '@/services/callService'

vi.mock('@/services/api', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  },
}))

import { apiClient } from '@/services/api'

const mockCall = {
  id: 'call-1',
  meeting_id: 'meeting-1',
  status: 'processing',
  created_at: '2026-01-01T00:00:00Z',
}

describe('callService', () => {
  beforeEach(() => vi.clearAllMocks())

  it('list() calls GET /calls and returns paginated data', async () => {
    const paginated = { data: [mockCall], total: 1, page: 1, page_size: 20 }
    vi.mocked(apiClient.get).mockResolvedValue({ data: paginated })

    const result = await callService.list()

    expect(apiClient.get).toHaveBeenCalledWith('/calls')
    expect(result.data).toEqual([mockCall])
  })

  it('list() appends query string when params are provided', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: { data: [], total: 0 } })

    await callService.list({ status: 'complete', sort: 'created_at_desc' })

    const url = vi.mocked(apiClient.get).mock.calls[0][0] as string
    expect(url).toContain('status=complete')
    expect(url).toContain('sort=created_at_desc')
  })

  it('getById() calls GET /calls/:id', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: { data: mockCall } })

    const result = await callService.getById('call-1')

    expect(apiClient.get).toHaveBeenCalledWith('/calls/call-1')
    expect(result.data).toEqual(mockCall)
  })

  it('create() calls POST /calls', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({ data: { data: mockCall } })

    const result = await callService.create({ meeting_id: 'meeting-1' })

    expect(apiClient.post).toHaveBeenCalledWith('/calls', { meeting_id: 'meeting-1' })
    expect(result.data).toEqual(mockCall)
  })

  it('update() calls PATCH /calls/:id', async () => {
    const updated = { ...mockCall, status: 'complete' }
    vi.mocked(apiClient.patch).mockResolvedValue({ data: { data: updated } })

    const result = await callService.update('call-1', { status: 'complete' })

    expect(apiClient.patch).toHaveBeenCalledWith('/calls/call-1', { status: 'complete' })
    expect(result.data).toEqual(updated)
  })

  it('delete() calls DELETE /calls/:id', async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({})

    await callService.delete('call-1')

    expect(apiClient.delete).toHaveBeenCalledWith('/calls/call-1')
  })
})
