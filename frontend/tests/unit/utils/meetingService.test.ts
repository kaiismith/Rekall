import { describe, it, expect, vi, beforeEach } from 'vitest'
import { meetingService } from '@/services/meetingService'

// Mock the apiClient.
vi.mock('@/services/api', () => ({
  apiClient: {
    post: vi.fn(),
    get: vi.fn(),
    delete: vi.fn(),
  },
}))

import { apiClient } from '@/services/api'

describe('meetingService', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('create calls POST /meetings', async () => {
    const mockMeeting = { id: '1', code: 'abc-defg-hij', type: 'open', status: 'waiting' }
    vi.mocked(apiClient.post).mockResolvedValue({ data: { success: true, data: mockMeeting } })

    const result = await meetingService.create({ type: 'open' })
    expect(apiClient.post).toHaveBeenCalledWith('/meetings', { type: 'open' })
    expect(result.data).toEqual(mockMeeting)
  })

  it('getByCode calls GET /meetings/:code', async () => {
    const mockMeeting = { id: '1', code: 'abc-defg-hij' }
    vi.mocked(apiClient.get).mockResolvedValue({ data: { success: true, data: mockMeeting } })

    const result = await meetingService.getByCode('abc-defg-hij')
    expect(apiClient.get).toHaveBeenCalledWith('/meetings/abc-defg-hij')
    expect(result.data).toEqual(mockMeeting)
  })

  it('listMine calls GET /meetings/mine with no params', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: { success: true, data: [] } })

    await meetingService.listMine()
    expect(apiClient.get).toHaveBeenCalledWith('/meetings/mine')
  })

  it('listMine passes filter[status] as query param', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: { success: true, data: [] } })

    await meetingService.listMine({ status: 'in_progress' })
    expect(apiClient.get).toHaveBeenCalledWith('/meetings/mine?filter%5Bstatus%5D=in_progress')
  })

  it('listMine passes sort as query param', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: { success: true, data: [] } })

    await meetingService.listMine({ sort: 'duration_desc' })
    expect(apiClient.get).toHaveBeenCalledWith('/meetings/mine?sort=duration_desc')
  })

  it('end calls DELETE /meetings/:code', async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({})
    await meetingService.end('abc-defg-hij')
    expect(apiClient.delete).toHaveBeenCalledWith('/meetings/abc-defg-hij')
  })

  it('buildWsUrl constructs a URL with the meeting code and token', () => {
    const url = meetingService.buildWsUrl('abc-defg-hij', 'my-token')
    expect(url).toContain('/api/v1/meetings/abc-defg-hij/ws')
    expect(url).toContain('token=my-token')
    // In test env VITE_API_BASE_URL is undefined so base is /api/v1 (no protocol prefix).
    // In production it will be a ws:// URL. Just assert the path and token are correct.
  })
})
