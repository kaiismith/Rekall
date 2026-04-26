export type MeetingType = 'open' | 'private'
export type MeetingStatus = 'waiting' | 'active' | 'ended'
export type MeetingScopeType = 'organization' | 'department'
export type ParticipantRole = 'host' | 'participant'
export type MeetingStatusFilter = 'in_progress' | 'complete' | 'processing' | 'failed'
export type MeetingSortKey =
  | 'created_at_desc'
  | 'created_at_asc'
  | 'duration_desc'
  | 'duration_asc'
  | 'title_asc'
  | 'title_desc'

export interface ParticipantPreview {
  user_id: string
  full_name: string
  initials: string
}

export interface Meeting {
  id: string
  code: string
  title: string
  type: MeetingType
  scope_type?: MeetingScopeType
  scope_id?: string
  host_id: string
  status: MeetingStatus
  max_participants: number
  /** Per-meeting toggle for the live-captions / ASR feature. Set by the host
   *  at creation. When false the captions UI is hidden. */
  transcription_enabled: boolean
  join_url: string
  started_at?: string
  ended_at?: string
  created_at: string
  duration_seconds?: number | null
  participant_previews?: ParticipantPreview[]
}

export interface ListMeetingsParams {
  status?: MeetingStatusFilter
  sort?: MeetingSortKey
}

export interface CreateMeetingPayload {
  title?: string
  type: MeetingType
  scope_type?: MeetingScopeType
  scope_id?: string
  /** Opt the meeting into live captions / transcription. Defaults to false. */
  transcription_enabled?: boolean
}

export interface MeetingParticipant {
  id: string
  meeting_id: string
  user_id: string
  role: ParticipantRole
  invited_by?: string
  joined_at?: string
  left_at?: string
}

// ─── WebSocket message types ──────────────────────────────────────────────────

export type WsMsgType =
  | 'offer'
  | 'answer'
  | 'ice_candidate'
  | 'speaking_state'
  | 'ping'
  | 'pong'
  | 'knock.requested'
  | 'knock.respond'
  | 'knock.approved'
  | 'knock.denied'
  | 'knock.resolved'
  | 'knock.cancelled'
  | 'participant.joined'
  | 'participant.left'
  | 'meeting.ended'
  // In-room controls
  | 'media_state'
  | 'force_mute'
  | 'emoji_reaction'
  | 'hand_raise'
  | 'room_state'
  | 'chat_message'
  | 'caption_chunk'

export interface WsMessage {
  type: WsMsgType
  // WebRTC / legacy
  from?: string
  to?: string
  knock_id?: string
  approved?: boolean
  user_id?: string
  payload?: unknown
  // In-room controls (server → client)
  audio?: boolean
  video?: boolean
  raised?: boolean
  emoji?: string
  from_id?: string          // emoji_reaction sender
  target_id?: string        // force_mute target (client → server)
  participants?: RoomStateParticipant[]
  // chat_message
  id?: string               // server-assigned message id (echo)
  client_id?: string        // client-generated optimistic id (echo)
  body?: string             // message body
  sent_at?: string          // ISO8601 server timestamp
  // Identity fields attached to participant.joined / chat broadcasts
  full_name?: string
  initials?: string
  // caption_chunk
  caption_kind?: 'partial' | 'final'
  caption_text?: string
  caption_segment_id?: string
  caption_ts?: number
}

/** A single caption entry in the merged meeting transcript feed. */
export interface CaptionEntry {
  /** Stable composite key: `${userId}:${segmentId}`. */
  key: string
  userId: string
  segmentId: string
  /** "partial" gets replaced when a "final" with the same key arrives. */
  kind: 'partial' | 'final'
  text: string
  /** Wall-clock ms since epoch when the chunk was produced. */
  timestamp: number
}

export interface KnockEntry {
  knock_id: string
  user_id: string
}

// State machine states for useMeeting hook
export type MeetingRoomState =
  // Pre-meeting: camera/mic preview screen — user picks devices and confirms
  // before the WebSocket is opened.
  | 'device_check'
  | 'connecting'
  | 'waiting_room'
  | 'in_meeting'
  | 'denied'
  | 'ended'
  | 'error'

// ─── In-room control types ────────────────────────────────────────────────────

/** Ephemeral audio/video state for a remote participant. */
export interface MediaState {
  audio: boolean
  video: boolean
}

/** A single floating emoji reaction in the reaction queue. */
export interface EmojiReaction {
  id: string        // local UUID for React key
  userId: string
  emoji: string
  timestamp: number
}

/** Snapshot entry sent by the server on join. */
export interface RoomStateParticipant {
  user_id: string
  full_name?: string
  initials?: string
  audio: boolean
  video: boolean
  hand_raised: boolean
}

/** Virtual background selection. */
export type BackgroundOption =
  | { type: 'none' }
  | { type: 'blur'; level: 'light' | 'heavy' }
  | { type: 'image'; src: string; label: string }

// ─── Chat types ───────────────────────────────────────────────────────────────

/** A single chat message in the in-room panel. */
export interface ChatMessage {
  /** Server id once confirmed; equal to `clientId` while pending or failed. */
  id: string
  /** Present on pending local sends; echoed back by the server for reconcile. */
  clientId?: string
  userId: string
  body: string
  /** Epoch milliseconds. */
  sentAt: number
  /** True between optimistic append and server echo. */
  pending?: boolean
  /** True when the server echo did not arrive within the timeout window. */
  failed?: boolean
}

/** Display entry for resolving a sender's name/avatar from a userId. */
export interface ParticipantDirectoryEntry {
  full_name: string
  initials: string
}

/** Response payload for GET /meetings/:code/messages. */
export interface ListChatMessagesResponse {
  messages: ChatMessage[]
  has_more: boolean
}
