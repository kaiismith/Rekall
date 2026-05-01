/** Kat: live AI notes assistant.
 *
 *  Notes are produced by the backend Kat scheduler against a sliding window
 *  of finalized transcript segments and broadcast over the meeting WS hub.
 *  They are NOT persisted server-side — the only durable record is the
 *  transcript itself. Late joiners receive the in-memory ring-buffer contents
 *  via SendToUser at join time on the same `kat.note` channel.
 */

export type KatStatus =
  | 'idle'
  | 'warming_up'
  | 'streaming' // OpenAI is producing tokens; partial is being typed live
  | 'live'
  | 'offline'
  | 'error'
  // Backend tick found no transcript segments in the window — nothing to
  // summarize. Panel renders a friendly "Nothing to take notes" message.
  | 'empty'

/** Wire-shape for a single note as broadcast over the meeting WS. */
export interface KatNoteDTO {
  id: string
  run_id: string
  meeting_id?: string
  call_id?: string
  window_started_at: string // RFC3339
  window_ended_at: string
  segment_index_lo?: number
  segment_index_hi?: number
  summary: string
  key_points: string[]
  open_questions: string[]
  model_id: string
  prompt_version: string
  /** Backend run outcome. "ok" is the normal summary; "empty_window" is a
   *  transient signal that the latest tick found no segments; "streaming"
   *  carries an in-flight partial response (Summary holds the running raw
   *  plain-text — the panel renders it as it grows). */
  status?: 'ok' | 'errored' | 'empty_window' | 'streaming'
}

/** Response shape of GET /healthz/kat. Top-level (not envelope-wrapped) so
 *  the frontend can probe it as a liveness signal without unwrapping. */
export interface KatHealthResponse {
  configured: boolean
  /** Selected backend: "foundry" (Azure AI Foundry) or "openai". Empty
   *  string when the backend has Kat unconfigured. */
  provider: 'foundry' | 'openai' | ''
  auth_mode: 'api_key' | 'managed_identity' | 'none'
  deployment: string
  endpoint_host: string
}
