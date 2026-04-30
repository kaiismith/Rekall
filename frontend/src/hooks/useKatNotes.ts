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
    setNotes((prev) => {
      // Dedupe by id; merge by `window_started_at` ascending.
      if (prev.some((n) => n.id === note.id)) return prev
      const merged = [...prev, note].sort((a, b) =>
        a.window_started_at.localeCompare(b.window_started_at),
      )
      // FIFO cap — drop the oldest entries when the array grows too long.
      if (merged.length > CLIENT_NOTE_CAP) {
        return merged.slice(merged.length - CLIENT_NOTE_CAP)
      }
      return merged
    })
    // First note flips warming_up -> live. Other states (offline, error,
    // idle) are sticky until probe() runs again.
    setStatus((s) => (s === 'warming_up' || s === 'idle' ? 'live' : s))
  }, [])

  const latestNote = useMemo(() => (notes.length > 0 ? notes[notes.length - 1] : null), [notes])

  // probe returns a Promise; the public `retry` contract is fire-and-forget.
  // eslint-disable-next-line @typescript-eslint/no-misused-promises
  return { notes, latestNote, status, health, pushNote, retry: probe }
}
