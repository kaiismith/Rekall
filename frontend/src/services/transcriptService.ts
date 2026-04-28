import { apiClient } from './api'
import type { ApiResponse } from '@/types/common'
import type {
  CallTranscriptResponse,
  MeetingTranscriptResponse,
  TranscriptSegmentRequest,
  TranscriptWordTiming,
} from '@/types/transcript'
import type { ASRFinalEvent } from '@/types/asr'

/**
 * Transcript persistence + read API.
 *
 * `postCallSegment` is called by the solo-call captions flow once per ASR
 * `final` event. It is intentionally fire-and-forget from the UI's
 * perspective: persistence failures must NEVER surface as a user-facing
 * error toast. The captions UX is the source of truth in real time; the
 * stored copy is the substrate for downstream insight extraction.
 */
export const transcriptService = {
  /**
   * Persist one `final` segment for a solo call. Resolves with `true` on
   * success and `false` on any failure (the call already logged a console
   * warning for diagnostics). Never throws.
   */
  async postCallSegment(callId: string, sessionId: string, event: ASRFinalEvent): Promise<boolean> {
    const body: TranscriptSegmentRequest = {
      segment_index: event.segment_id,
      text: event.text,
      language: event.language || undefined,
      confidence: event.confidence,
      start_ms: event.start_ms,
      end_ms: event.end_ms,
      words: event.words as TranscriptWordTiming[] | undefined,
    }
    try {
      await apiClient.post(`/calls/${callId}/asr-session/${sessionId}/segments`, body)
      return true
    } catch (err) {
      // Diagnostic only — never surfaced to the user. The captions UX is
      // unaffected; only the stored copy is degraded.
      console.warn('transcript: postCallSegment failed', {
        sessionId,
        segment_id: event.segment_id,
        err,
      })
      return false
    }
  },

  /** Read the persisted transcript for a solo call. */
  async getCallTranscript(callId: string): Promise<CallTranscriptResponse> {
    const response = await apiClient.get<ApiResponse<CallTranscriptResponse>>(
      `/calls/${callId}/transcript`,
    )
    return response.data.data
  },

  /** Read the persisted transcript for a meeting (all participants' sessions). */
  async getMeetingTranscript(meetingCode: string): Promise<MeetingTranscriptResponse> {
    const response = await apiClient.get<ApiResponse<MeetingTranscriptResponse>>(
      `/meetings/${meetingCode}/transcript`,
    )
    return response.data.data
  },
}
