import { apiClient } from './api'
import type { ApiResponse } from '@/types/common'
import type {
  Meeting,
  CreateMeetingPayload,
  ListMeetingsParams,
  ChatMessage,
  ListChatMessagesResponse,
} from '@/types/meeting'

/** Raw chat message shape returned by the backend (snake_case, ISO timestamp). */
interface RawChatMessage {
  id: string
  meeting_id: string
  user_id: string
  body: string
  sent_at: string
}

/** Raw chat history response envelope. */
interface RawChatListData {
  messages: RawChatMessage[]
  has_more: boolean
}

function normaliseChatMessage(raw: RawChatMessage): ChatMessage {
  return {
    id: raw.id,
    userId: raw.user_id,
    body: raw.body,
    sentAt: new Date(raw.sent_at).getTime(),
  }
}

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
   * Fetch the chat message history for a meeting.
   * `before` is an RFC3339 cursor — when set, only messages strictly older
   * than that timestamp are returned. `limit` defaults to 50 server-side and
   * is clamped to [1, 100].
   */
  async listMessages(
    code: string,
    params?: { before?: string; limit?: number },
  ): Promise<ListChatMessagesResponse> {
    const qs = new URLSearchParams()
    if (params?.before) qs.set('before', params.before)
    if (params?.limit != null) qs.set('limit', String(params.limit))
    const query = qs.toString() ? `?${qs.toString()}` : ''
    const response = await apiClient.get<ApiResponse<RawChatListData>>(
      `/meetings/${code}/messages${query}`,
    )
    const { messages, has_more } = response.data.data
    return {
      messages: messages.map(normaliseChatMessage),
      has_more,
    }
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
