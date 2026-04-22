import { apiClient } from './api'
import type { ApiResponse } from '@/types/common'
import type { Meeting, CreateMeetingPayload, ListMeetingsParams } from '@/types/meeting'

export const meetingService = {
  /** Create a new meeting. */
  async create(payload: CreateMeetingPayload): Promise<ApiResponse<Meeting>> {
    const response = await apiClient.post<ApiResponse<Meeting>>('/meetings', payload)
    return response.data
  },

  /** Fetch a meeting by join code. */
  async getByCode(code: string): Promise<ApiResponse<Meeting>> {
    const response = await apiClient.get<ApiResponse<Meeting>>(`/meetings/${code}`)
    return response.data
  },

  /** List meetings where the current user is host or participant. */
  async listMine(params?: ListMeetingsParams): Promise<ApiResponse<Meeting[]>> {
    const qs = new URLSearchParams()
    if (params?.status) qs.set('filter[status]', params.status)
    if (params?.sort) qs.set('sort', params.sort)
    const query = qs.toString() ? `?${qs.toString()}` : ''
    const response = await apiClient.get<ApiResponse<Meeting[]>>(`/meetings/mine${query}`)
    return response.data
  },

  /** End a meeting (host only). */
  async end(code: string): Promise<void> {
    await apiClient.delete(`/meetings/${code}`)
  },

  /**
   * Build the WebSocket URL for a meeting. The access token is passed as a
   * query parameter because WebSocket clients cannot set custom headers.
   */
  buildWsUrl(code: string, accessToken: string): string {
    const apiBase = (import.meta.env['VITE_API_BASE_URL'] as string | undefined) ?? '/api/v1'
    const wsBase = apiBase.replace(/^http/, 'ws').replace(/\/api\/v1\/?$/, '')
    return `${wsBase}/api/v1/meetings/${code}/ws?token=${encodeURIComponent(accessToken)}`
  },
}
