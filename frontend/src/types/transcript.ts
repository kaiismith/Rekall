/**
 * Wire shapes for the persisted transcript endpoints
 *
 *   POST /api/v1/calls/:id/asr-session/:session_id/segments
 *   GET  /api/v1/calls/:id/transcript
 *   GET  /api/v1/meetings/:code/transcript
 *
 * Mirrors the backend DTOs in [transcript_dto.go](../../../backend/internal/interfaces/http/dto/transcript_dto.go).
 */

import type { EngineMode } from './asr'

/**
 * Per-word timing as stored in transcript_segments.words (JSONB).
 * Field tags match the ASR proto so the JSONB round-trip is faithful.
 */
export interface TranscriptWordTiming {
  w: string
  start_ms: number
  end_ms: number
  p: number
}

/**
 * Body for POST /api/v1/calls/:id/asr-session/:session_id/segments.
 * Identical in shape to the meeting-WS caption payload's persistence fields,
 * minus the relay-only fields (caption_kind, caption_segment_id, caption_ts).
 */
export interface TranscriptSegmentRequest {
  segment_index: number
  text: string
  language?: string
  confidence?: number
  start_ms: number
  end_ms: number
  words?: TranscriptWordTiming[]
}

export interface TranscriptSessionDTO {
  id: string
  speaker_user_id: string
  call_id?: string
  meeting_id?: string
  engine_mode: EngineMode | 'legacy'
  model_id: string
  language_requested?: string
  status: 'active' | 'ended' | 'errored' | 'expired'
  started_at: string // ISO-8601
  ended_at?: string
  finalized_segment_count: number
  audio_seconds_total: number
}

export interface TranscriptSegmentDTO {
  id: string
  session_id: string
  segment_index: number
  speaker_user_id: string
  text: string
  language?: string
  confidence?: number
  start_ms: number
  end_ms: number
  words?: TranscriptWordTiming[]
  engine_mode: EngineMode | 'legacy'
  model_id: string
  segment_started_at: string // ISO-8601
}

/**
 * Page-window metadata returned alongside paginated transcript reads.
 * Matches dto.TranscriptPagination on the backend.
 */
export interface TranscriptPagination {
  page: number
  per_page: number
  total: number
  total_pages: number
  has_more: boolean
}

export interface CallTranscriptResponse {
  session?: TranscriptSessionDTO
  segments: TranscriptSegmentDTO[]
  pagination: TranscriptPagination
}

export interface MeetingTranscriptResponse {
  sessions: TranscriptSessionDTO[]
  segments: TranscriptSegmentDTO[]
  pagination: TranscriptPagination
}
