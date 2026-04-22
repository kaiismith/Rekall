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
  | 'laser_move'
  | 'laser_stop'
  | 'room_state'

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
  x?: number
  y?: number
  emoji?: string
  from_id?: string          // emoji_reaction sender
  target_id?: string        // force_mute target (client → server)
  participants?: RoomStateParticipant[]
}

export interface KnockEntry {
  knock_id: string
  user_id: string
}

// State machine states for useMeeting hook
export type MeetingRoomState =
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

/** Current laser pointer position (null when no laser is active). */
export interface LaserState {
  userId: string
  x: number   // 0–1 normalised
  y: number   // 0–1 normalised
}

/** Snapshot entry sent by the server on join. */
export interface RoomStateParticipant {
  user_id: string
  audio: boolean
  video: boolean
  hand_raised: boolean
  laser_active: boolean
}

/** Virtual background selection. */
export type BackgroundOption =
  | { type: 'none' }
  | { type: 'blur'; level: 'light' | 'heavy' }
  | { type: 'image'; src: string; label: string }
