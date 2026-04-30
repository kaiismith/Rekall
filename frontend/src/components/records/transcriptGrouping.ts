import type { TranscriptSegmentDTO } from '@/types/transcript'

export interface SpeakerBlock {
  speakerId: string
  startMs: number
  segments: TranscriptSegmentDTO[]
}

/**
 * Reduce a flat segment slice into speaker-grouped blocks. Consecutive
 * same-speaker segments are merged into a single block; a speaker change
 * starts a new one. Pure / exported for unit testing.
 */
export function groupBySpeaker(segments: TranscriptSegmentDTO[]): SpeakerBlock[] {
  const blocks: SpeakerBlock[] = []
  for (const seg of segments) {
    const last = blocks[blocks.length - 1]
    if (last && last.speakerId === seg.speaker_user_id) {
      last.segments.push(seg)
    } else {
      blocks.push({
        speakerId: seg.speaker_user_id,
        startMs: seg.start_ms,
        segments: [seg],
      })
    }
  }
  return blocks
}

/**
 * Format an offset-from-session-start in milliseconds to `M:SS` (or
 * `H:MM:SS` once the offset crosses 1 hour).
 */
export function formatTimestamp(ms: number): string {
  const totalSeconds = Math.max(0, Math.floor(ms / 1000))
  const h = Math.floor(totalSeconds / 3600)
  const m = Math.floor((totalSeconds % 3600) / 60)
  const s = totalSeconds % 60
  if (h > 0) {
    return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
  }
  return `${m}:${String(s).padStart(2, '0')}`
}
