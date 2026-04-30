/** Kat: live AI notes assistant.
 *
 *  Notes are produced by the backend Kat scheduler against a sliding window
 *  of finalized transcript segments and broadcast over the meeting WS hub.
 *  They are NOT persisted server-side — the only durable record is the
 *  transcript itself. Late joiners receive the in-memory ring-buffer contents
 *  via SendToUser at join time on the same `kat.note` channel.
 */

export type KatStatus = 'idle' | 'warming_up' | 'live' | 'offline' | 'error'

/** Wire-shape for a single note as broadcast over the meeting WS. */
export interface KatNoteDTO {
  id: string
  run_id: string
  meeting_id?: string
  call_id?: string
  window_started_at: string // RFC3339
  window_ended_at: string
  segment_index_lo: number
  segment_index_hi: number
  summary: string
  key_points: string[]
  open_questions: string[]
  model_id: string
  prompt_version: string
}

/** Response shape of GET /healthz/kat. Top-level (not envelope-wrapped) so
 *  the frontend can probe it as a liveness signal without unwrapping. */
export interface KatHealthResponse {
  configured: boolean
  auth_mode: 'api_key' | 'managed_identity' | 'none'
  deployment: string
  endpoint_host: string
}
