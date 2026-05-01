import { apiClient } from './api'
import type { ApiResponse } from '@/types/common'
import type {
  Meeting,
  CreateMeetingPayload,
  ListMeetingsParams,
  PaginatedMeetingListResponse,
  ChatMessage,
  ListChatMessagesResponse,
} from '@/types/meeting'
import type { Scope } from '@/types/scope'
import { scopeToQueryParams } from '@/utils/scope'

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

  /**
   * List meetings the current user can see, paginated. By default scoped to
   * host + participant; pass `scope` to filter to a specific organization,
   * department, or open-only slice. Pass `params.page` / `params.per_page`
   * to fetch a specific page (defaults: page 1, per_page 20).
   */
  async listMine(
    params?: ListMeetingsParams,
    scope?: Scope | null,
  ): Promise<PaginatedMeetingListResponse> {
    const qs = new URLSearchParams()
    if (params?.status) qs.set('filter[status]', params.status)
    if (params?.sort) qs.set('sort', params.sort)
    if (params?.page) qs.set('page', String(params.page))
    if (params?.per_page) qs.set('per_page', String(params.per_page))
    const scopeParams = scopeToQueryParams(scope ?? null)
    for (const [k, v] of Object.entries(scopeParams)) {
      if (v != null) qs.set(k, v)
    }
    const query = qs.toString() ? `?${qs.toString()}` : ''
    const response = await apiClient.get<PaginatedMeetingListResponse>(`/meetings/mine${query}`)
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
   * Request a short-lived, single-use ticket for opening the meeting
   * signaling WebSocket. The bearer token is attached by the axios request
   * interceptor; the returned ticket is opaque, carries its own 60-second
   * TTL, and is consumed atomically on the WS upgrade.
   */
  async requestWsTicket(
    code: string,
  ): Promise<{ ticket: string; wsUrl: string; expiresAt: number }> {
    const response = await apiClient.post<
      ApiResponse<{
        ticket: string
        expires_at: string
        ws_url: string
      }>
    >(`/meetings/${code}/ws-ticket`)
    const d = response.data.data
    return {
      ticket: d.ticket,
      wsUrl: d.ws_url,
      expiresAt: new Date(d.expires_at).getTime(),
    }
  },

  /**
   * Build the absolute WebSocket URL from a server-provided relative `wsUrl`
   * path (returned by requestWsTicket). The backend returns a path starting
   * with `/api/v1/…`; we derive the ws:// origin from the configured API base.
   */
  buildAbsoluteWsUrl(wsUrl: string): string {
    const apiBase = (import.meta.env['VITE_API_BASE_URL'] as string | undefined) ?? '/api/v1'
    const origin = apiBase.replace(/^http/, 'ws').replace(/\/api\/v1\/?$/, '')
    return `${origin}${wsUrl}`
  },
}
