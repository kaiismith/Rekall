import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

import { katService } from '@/services/katService'
import type { KatHealthResponse, KatNoteDTO, KatStatus } from '@/types/kat'

/** Maximum number of notes the hook keeps client-side. Matches the backend
 *  ring-buffer capacity default (KAT_RING_BUFFER_CAPACITY=20). */
const CLIENT_NOTE_CAP = 20

/** How long the hook waits in `warming_up` before silently staying there.
 *  No state transition fires; the panel just keeps rendering "Kat is
 *  listening…". The user explicitly requested no auto-flip-to-error on
 *  silence — silence is normal at the start of a meeting. */
const WARMING_UP_PATIENCE_MS = 60_000

export interface UseKatNotesResult {
  notes: KatNoteDTO[]
  latestNote: KatNoteDTO | null
  status: KatStatus
  /** Health probe response; used by the panel footer ("model: gpt-4o-mini"). */
  health: KatHealthResponse | null
  /** Running raw text from an in-flight streaming response. Cleared when
   *  the next final ('ok') note arrives. The panel renders this directly
   *  while the LLM is producing tokens. */
  streamingPartial: string | null
  /** Push a note received over the meeting WS. Idempotent on `id`. */
  pushNote: (note: KatNoteDTO) => void
  /** Manual retry — re-runs the bootstrap probe. */
  retry: () => void
}

/** Live AI notes hook.
 *
 *  Bootstraps from `/healthz/kat`, then waits for `kat.note` WS messages to
 *  arrive via `pushNote`. The first incoming note flips `status` from
 *  `warming_up` to `live`. The hook deduplicates by `id` and caps the array
 *  client-side at `CLIENT_NOTE_CAP` (FIFO eviction).
 *
 *  When `/healthz/kat` reports `configured=false`, status goes to `offline`
 *  and the panel renders the offline card. The hook NEVER throws.
 */
export function useKatNotes(): UseKatNotesResult {
  const [notes, setNotes] = useState<KatNoteDTO[]>([])
  const [status, setStatus] = useState<KatStatus>('idle')
  const [health, setHealth] = useState<KatHealthResponse | null>(null)
  // Latest in-flight streaming text. Reset on each completed note. The
  // panel renders this directly during the typing animation.
  const [streamingPartial, setStreamingPartial] = useState<string | null>(null)
  const warmingUpTimerRef = useRef<number | null>(null)
  const probeIDRef = useRef(0)

  const probe = useCallback(async () => {
    const id = ++probeIDRef.current
    setStatus('idle')
    try {
      const h = await katService.getHealth()
      // Guard against an out-of-order resolve from a previous probe call.
      if (id !== probeIDRef.current) return
      setHealth(h)
      if (!h.configured) {
        setStatus('offline')
        return
      }
      setStatus('warming_up')
    } catch {
      // katService swallows errors; this branch is defensive.
      if (id !== probeIDRef.current) return
      setStatus('error')
    }
  }, [])

  useEffect(() => {
    void probe()
    return () => {
      if (warmingUpTimerRef.current !== null) {
        window.clearTimeout(warmingUpTimerRef.current)
      }
    }
  }, [probe])

  // Stay-in-warming-up timeout: just clears the timer ref so re-mounts don't
  // leak. We do NOT auto-flip to `error` — silence is expected on a quiet
  // meeting and would surface as an inappropriate red state on the panel.
  useEffect(() => {
    if (status !== 'warming_up') return
    if (warmingUpTimerRef.current !== null) {
      window.clearTimeout(warmingUpTimerRef.current)
    }
    warmingUpTimerRef.current = window.setTimeout(() => {
      warmingUpTimerRef.current = null
    }, WARMING_UP_PATIENCE_MS)
  }, [status])

  const pushNote = useCallback((note: KatNoteDTO) => {
    // Diagnostic — verbose on purpose. Lets the operator see in DevTools
    // exactly what the WS delivered for each kat.note message. Pre-shipping
    // we can switch this to a debug-only flag, but during the live tuning
    // pass it's invaluable.
    // eslint-disable-next-line no-console
    console.debug('[kat] note received', {
      status: note.status,
      id: note.id,
      run_id: note.run_id,
      summary_preview: note.summary?.slice(0, 200),
      summary_len: note.summary?.length ?? 0,
      key_points: note.key_points,
      open_questions: note.open_questions,
    })

    // Streaming partial: in-flight LLM response. Update the running buffer
    // so the panel can show tokens as they arrive. Don't push to the
    // notes list — it's intermediate state.
    if (note.status === 'streaming') {
      setStreamingPartial(note.summary)
      setStatus((s) => (s === 'offline' || s === 'error' ? s : 'streaming'))
      return
    }

    // Empty-window: backend tick found no transcript segments. Don't add
    // the placeholder to the notes list; just flip status so the panel
    // shows the empty-state copy.
    if (note.status === 'empty_window') {
      setStatus((s) => {
        if (s === 'offline' || s === 'error') return s
        if (s === 'live') return s
        return 'empty'
      })
      return
    }

    // Errored notes: log only, don't disturb the panel.
    if (note.status === 'errored') {
      console.warn('kat: errored note received', note)
      return
    }

    // Final ('ok') note: handoff to the structured "live" view, but give
    // the typewriter ~700ms grace so the panel doesn't snap from
    // mid-typing to the final layout.
    //
    // We render whatever the model produced — including short or
    // boilerplate summaries. If the model collapses substantive content
    // into a placeholder, that's a prompt-tuning issue to fix at the
    // source. Hiding output here would mask the bug.
    setNotes((prev) => {
      if (prev.some((n) => n.id === note.id)) return prev
      const merged = [...prev, note].sort((a, b) =>
        a.window_started_at.localeCompare(b.window_started_at),
      )
      if (merged.length > CLIENT_NOTE_CAP) {
        return merged.slice(merged.length - CLIENT_NOTE_CAP)
      }
      return merged
    })
    // Keep the streaming view visible briefly so the typewriter can finish.
    window.setTimeout(() => {
      setStreamingPartial(null)
      setStatus((s) => {
        if (s === 'offline' || s === 'error') return s
        return 'live'
      })
    }, 700)
  }, [])

  const latestNote = useMemo(() => (notes.length > 0 ? notes[notes.length - 1] : null), [notes])

  // probe returns a Promise; the public `retry` contract is fire-and-forget.
  // eslint-disable-next-line @typescript-eslint/no-misused-promises
  return { notes, latestNote, status, health, streamingPartial, pushNote, retry: probe }
}
