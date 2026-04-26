import { apiClient } from './api'
import type { ApiResponse } from '@/types/common'
import type {
  ASRSessionPayload,
  ASRSessionEndPayload,
  ASRSessionRequest,
} from '@/types/asr'

/**
 * ASR session API.
 *
 * The session_token returned by `request()` is single-use and bound to a
 * specific call. Callers MUST NOT log the token; only the session_id and
 * request status appear in any frontend telemetry.
 */
export const asrService = {
  /** Register an ASR session for `callId` and obtain the WebSocket URL. */
  async request(callId: string, payload: ASRSessionRequest = {}): Promise<ASRSessionPayload> {
    const response = await apiClient.post<ApiResponse<ASRSessionPayload>>(
      `/calls/${callId}/asr-session`,
      payload,
    )
    return response.data.data
  },

  /** Tell the ASR service to terminate `sessionId` and return the final transcript. */
  async end(callId: string, sessionId: string): Promise<ASRSessionEndPayload> {
    const response = await apiClient.post<ApiResponse<ASRSessionEndPayload>>(
      `/calls/${callId}/asr-session/end`,
      { session_id: sessionId },
    )
    return response.data.data
  },

  /**
   * Register an ASR session against a meeting (rather than a call). The
   * caller MUST already be an active participant of the meeting AND the
   * meeting MUST have `transcription_enabled = true`. Returns 403
   * TRANSCRIPTION_DISABLED otherwise.
   */
  async requestForMeeting(meetingCode: string, payload: ASRSessionRequest = {}): Promise<ASRSessionPayload> {
    const response = await apiClient.post<ApiResponse<ASRSessionPayload>>(
      `/meetings/${meetingCode}/asr-session`,
      payload,
    )
    return response.data.data
  },

  /** End the meeting-scoped ASR session. Idempotent. */
  async endForMeeting(meetingCode: string, sessionId: string): Promise<ASRSessionEndPayload> {
    const response = await apiClient.post<ApiResponse<ASRSessionEndPayload>>(
      `/meetings/${meetingCode}/asr-session/end`,
      { session_id: sessionId },
    )
    return response.data.data
  },

  /**
   * URL the AudioWorklet processor is loaded from. Served as a static
   * asset under public/ so dev and production resolve to the same path
   * without bundler involvement — `new URL('…ts', import.meta.url)` does
   * NOT survive Vite's prod build for AudioWorklet purposes (the URL ends
   * up pointing at a non-existent .ts file and addModule() throws).
   */
  buildAudioWorkletUrl(): string {
    return '/pcm-worklet.js'
  },
}
