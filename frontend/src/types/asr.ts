/**
 * ASR session DTOs and discriminated WebSocket event types.
 *
 * Wire format mirrors the C++ rekall-asr service. Audio is sent as raw binary
 * frames (16-kHz int16 LE mono); transcript events arrive as JSON text frames.
 */

export interface ASRSessionPayload {
  session_id: string
  session_token: string
  ws_url: string
  expires_at: string // ISO-8601
  model_id: string
  sample_rate: number
  frame_format: 'pcm_s16le_mono'
}

export interface ASRSessionEndPayload {
  final_transcript: string
  final_count: number
}

export interface ASRSessionRequest {
  model_id?: string
  language?: string
  ttl_seconds?: number
}

// ── Server → client events ───────────────────────────────────────────────────

/**
 * Engine running on the asr service. Surfaced so the captions UI can render a
 * "Cloud" / "Local" badge and skip the partial placeholder when the engine is
 * one-shot. Optional for backward compat with older servers.
 */
export type EngineMode = 'local' | 'openai'

export interface ASRReadyEvent {
  type: 'ready'
  session_id: string
  model_id: string
  sample_rate: number
  engine_mode?: EngineMode
}

export interface ASRPartialEvent {
  type: 'partial'
  segment_id: number
  text: string
  start_ms: number
  end_ms: number
  confidence: number
}

export interface ASRWordTiming {
  w: string
  start_ms: number
  end_ms: number
  p: number
}

export interface ASRFinalEvent {
  type: 'final'
  segment_id: number
  text: string
  language: string
  start_ms: number
  end_ms: number
  /** Aggregate confidence in [0,1]; absent when the engine doesn't report it. */
  confidence?: number
  words?: ASRWordTiming[]
}

export interface ASRInfoEvent {
  type: 'info'
  code: string
  message?: string
}

export interface ASRErrorEvent {
  type: 'error'
  code: string
  message: string
}

export interface ASRPongEvent {
  type: 'pong'
  ts: number
}

export type ASRServerEvent =
  | ASRReadyEvent
  | ASRPartialEvent
  | ASRFinalEvent
  | ASRInfoEvent
  | ASRErrorEvent
  | ASRPongEvent

// ── Client → server control frames ──────────────────────────────────────────

export type ASRClientControl =
  | { type: 'config'; language?: string; translate?: boolean }
  | { type: 'flush' }
  | { type: 'ping' }

// ── Hook state ─────────────────────────────────────────────────────────────

export type ASRHookState =
  | 'idle'
  | 'requesting'
  | 'connecting'
  | 'streaming'
  | 'reconnecting'
  | 'ended'
  | 'error'
