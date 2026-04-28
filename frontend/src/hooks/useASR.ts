import { useCallback, useEffect, useRef, useState } from 'react'

import { asrService } from '@/services/asrService'
import type {
  ASRFinalEvent,
  ASRHookState,
  ASRServerEvent,
  ASRSessionPayload,
  EngineMode,
} from '@/types/asr'

/**
 * useASR — orchestrates a live captions session for `callId`.
 *
 * State machine:
 *   idle → requesting → connecting → streaming → ended
 *                ↘                ↘ unexpected close → reconnecting → requesting
 *                ↘ ticket fail → error
 *
 * `partial` is the rolling text for the active segment (replaced by every
 * new partial event); `finals` is the append-only list of frozen segments.
 */

export interface UseASRResult {
  state: ASRHookState
  partial: string
  finals: ASRFinalEvent[]
  error: { code: string; message: string } | null
  /**
   * Engine reported by the asr service in its `ready` event. `undefined` when
   * the server hasn't sent the field (older asr build) — older clients keep
   * working as before, just without the badge / placeholder UI.
   */
  engineMode: EngineMode | undefined
  /**
   * The active ASR session_id once the server has issued a token. Exposed
   * so callers can wire `onFinalSegment` to a per-segment POST without
   * threading the session through their own state. `null` while idle /
   * before the first `request()` returns.
   */
  sessionId: string | null
  start: () => Promise<void>
  stop: () => Promise<void>
}

/**
 * Optional per-chunk hooks. The meeting flow uses these to relay each
 * partial / final to the meeting WebSocket so other participants can render
 * the local speaker's transcript with attribution. `segmentId` is stable
 * across the partial→final transition for a single utterance, allowing the
 * receiving side to replace the in-flight partial in place rather than
 * appending duplicates.
 */
export interface ASRStreamCallbacks {
  onPartial?: (text: string, segmentId: string) => void
  onFinal?: (text: string, segmentId: string) => void
  /**
   * Fired exactly once per `final` event, with the FULL ASR payload (text +
   * timing + per-word + language + confidence). Used by the persistence
   * flows: the meeting page wires this to the WS hub's caption_chunk
   * relay (which both broadcasts and persists), the solo-call page wires
   * it to a direct POST to /asr-session/:sid/segments. Never fires for
   * `partial` events — partials are not persisted.
   */
  onFinalSegment?: (event: ASRFinalEvent) => void
}

const RECONNECT_BACKOFFS_MS = [500, 1000, 2000, 4000, 8000]
const MAX_RECONNECTS = RECONNECT_BACKOFFS_MS.length

function logSafe(message: string, data?: Record<string, unknown>): void {
  // Token / audio bytes MUST NOT appear in any console output. Callers pass
  // only `session_id`, request status, and similar non-sensitive metadata.
  // eslint-disable-next-line no-console
  console.info(`[asr] ${message}`, data ?? {})
}

/**
 * Identifies which backend endpoint pair the hook should use:
 *  - `call`    → /calls/:id/asr-session         (caller must own the call)
 *  - `meeting` → /meetings/:code/asr-session    (caller must be an active
 *                 participant AND meeting.transcription_enabled must be true)
 *
 * The wire protocol (WebSocket frames, JWT shape, transcript events) is
 * identical across both — only the registration endpoint differs.
 */
export type ASRSessionKind = 'call' | 'meeting'

export function useASR(
  id: string | null,
  kind: ASRSessionKind = 'call',
  callbacks?: ASRStreamCallbacks,
): UseASRResult {
  const [state, setState] = useState<ASRHookState>('idle')
  const [partial, setPartial] = useState('')
  const [finals, setFinals] = useState<ASRFinalEvent[]>([])
  const [error, setError] = useState<{ code: string; message: string } | null>(null)
  const [engineMode, setEngineMode] = useState<EngineMode | undefined>(undefined)
  const [sessionId, setSessionId] = useState<string | null>(null)

  const wsRef = useRef<WebSocket | null>(null)
  const audioCtxRef = useRef<AudioContext | null>(null)
  const workletNodeRef = useRef<AudioWorkletNode | null>(null)
  const mediaStreamRef = useRef<MediaStream | null>(null)
  const sessionRef = useRef<ASRSessionPayload | null>(null)
  const reconnectsRef = useRef(0)
  const stoppedRef = useRef(false)

  // Hold the latest callbacks in a ref so we don't re-create openSession on
  // every parent render — the callbacks themselves are typically inline
  // arrows that change identity each render.
  const callbacksRef = useRef<ASRStreamCallbacks | undefined>(callbacks)
  callbacksRef.current = callbacks

  // ── Cleanup of audio + WS resources ────────────────────────────────────────
  const teardown = useCallback(async () => {
    if (workletNodeRef.current) {
      workletNodeRef.current.disconnect()
      workletNodeRef.current = null
    }
    if (audioCtxRef.current) {
      try {
        await audioCtxRef.current.close()
      } catch {
        /* ignore */
      }
      audioCtxRef.current = null
    }
    if (mediaStreamRef.current) {
      mediaStreamRef.current.getTracks().forEach((t) => t.stop())
      mediaStreamRef.current = null
    }
    if (wsRef.current) {
      try {
        wsRef.current.close(1000, 'client teardown')
      } catch {
        /* ignore */
      }
      wsRef.current = null
    }
  }, [])

  // ── Open the ws + acquire mic + start the worklet ─────────────────────────
  const openSession = useCallback(async (): Promise<void> => {
    if (!id) return
    setError(null)
    setState('requesting')

    let session: ASRSessionPayload
    try {
      session =
        kind === 'meeting'
          ? await asrService.requestForMeeting(id, {})
          : await asrService.request(id, {})
      sessionRef.current = session
      setSessionId(session.session_id)
      logSafe('session issued', {
        kind,
        session_id: session.session_id,
        model_id: session.model_id,
      })
    } catch (e) {
      const err = normaliseError(e)
      setError(err)
      setState('error')
      return
    }

    setState('connecting')
    const ws = new WebSocket(session.ws_url)
    ws.binaryType = 'arraybuffer'
    wsRef.current = ws

    ws.onopen = async () => {
      try {
        const ctx = new AudioContext({ sampleRate: 16000 })
        audioCtxRef.current = ctx
        await ctx.audioWorklet.addModule(asrService.buildAudioWorkletUrl())

        const stream = await navigator.mediaDevices.getUserMedia({
          audio: {
            sampleRate: 16000,
            channelCount: 1,
            echoCancellation: true,
            noiseSuppression: true,
            autoGainControl: true,
          },
        })
        mediaStreamRef.current = stream

        const src = ctx.createMediaStreamSource(stream)
        const node = new AudioWorkletNode(ctx, 'pcm-worklet')
        workletNodeRef.current = node
        node.port.onmessage = (ev: MessageEvent<ArrayBuffer>) => {
          if (ws.readyState === WebSocket.OPEN) ws.send(ev.data)
        }
        src.connect(node)
        // Worklet must be connected somewhere for `process` to fire even
        // though we don't want any monitoring playback.
        node.connect(ctx.destination)

        reconnectsRef.current = 0
        setState('streaming')
        logSafe('streaming started')
      } catch (e) {
        const err = normaliseError(e)
        // Surface the actual cause to the console — the captions panel only
        // shows a short message; failures here (worklet load, mic permission)
        // are otherwise invisible. Audio bytes never appear in this code path,
        // so logging is safe.

        console.error('[asr] onopen failed', err, e)
        setError(err)
        setState('error')
        await teardown()
      }
    }

    ws.onmessage = (ev: MessageEvent<string>) => {
      let parsed: ASRServerEvent
      try {
        parsed = JSON.parse(ev.data) as ASRServerEvent
      } catch {
        return
      }
      switch (parsed.type) {
        case 'ready':
          // Pick up the engine the asr service is running. Optional field;
          // older servers don't send it and engineMode stays undefined.
          if (parsed.engine_mode === 'local' || parsed.engine_mode === 'openai') {
            setEngineMode(parsed.engine_mode)
          }
          break
        case 'partial':
          // eslint-disable-next-line no-console
          console.debug('[asr] ⟵ partial', {
            seg: parsed.segment_id,
            text: parsed.text,
            len: parsed.text.length,
          })
          setPartial(parsed.text)
          callbacksRef.current?.onPartial?.(parsed.text, String(parsed.segment_id))
          break
        case 'final':
          // eslint-disable-next-line no-console
          console.debug('[asr] ⟵ FINAL ', {
            seg: parsed.segment_id,
            text: parsed.text,
            len: parsed.text.length,
            startMs: parsed.start_ms,
            endMs: parsed.end_ms,
          })
          setPartial('')
          setFinals((cur) => [...cur, parsed])
          callbacksRef.current?.onFinal?.(parsed.text, String(parsed.segment_id))
          // Full-payload variant for the persistence flows (Task 10 of the
          // transcript-persistence spec). Fires synchronously, exactly once.
          callbacksRef.current?.onFinalSegment?.(parsed)
          break
        case 'error':
          console.warn('[asr] ⟵ error', parsed.code, parsed.message)
          setError({ code: parsed.code, message: parsed.message })
          break
        default:
          // eslint-disable-next-line no-console
          console.debug('[asr] ⟵ event', parsed)
          break
      }
    }

    ws.onclose = (ev: CloseEvent) => {
      if (stoppedRef.current) {
        setState('ended')
        return
      }
      // 1000 (normal), 4401 (auth), 4412 (hard timeout) are terminal.
      if (ev.code === 1000 || ev.code === 4401 || ev.code === 4412) {
        setState('ended')
        return
      }
      // Otherwise attempt reconnect.
      void scheduleReconnect()
    }

    ws.onerror = () => {
      // No payload — `onclose` fires next with a code we can act on.
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, kind, teardown])

  // ── Reconnect with capped exponential backoff ──────────────────────────────
  const scheduleReconnect = useCallback(async (): Promise<void> => {
    const attempt = reconnectsRef.current
    if (attempt >= MAX_RECONNECTS) {
      setError({ code: 'ASR_RECONNECT_EXHAUSTED', message: 'asr connection lost' })
      setState('error')
      await teardown()
      return
    }
    const wait = RECONNECT_BACKOFFS_MS[attempt]
    reconnectsRef.current = attempt + 1
    setState('reconnecting')
    logSafe('reconnecting', { attempt: attempt + 1, wait_ms: wait })
    await teardown()
    setTimeout(() => {
      if (!stoppedRef.current) void openSession()
    }, wait)
  }, [openSession, teardown])

  // ── Public API ─────────────────────────────────────────────────────────────
  const start = useCallback(async () => {
    // Idempotent: if a session is already in flight or running, do nothing.
    // The parent intent-driven effect can fire start() multiple times under
    // React 18 strict-mode double-invocation or rapid input flips; without
    // this guard we'd stack multiple sessions on top of each other.
    if (wsRef.current || sessionRef.current) return
    stoppedRef.current = false
    reconnectsRef.current = 0
    setFinals([])
    setPartial('')
    await openSession()
  }, [openSession])

  const stop = useCallback(async () => {
    // Nothing to tear down — the parent intent effect calls stop() on mount
    // and on every "captions off" transition; bail before mutating state so
    // we don't churn an idle hook into 'ended'.
    if (!wsRef.current && !sessionRef.current) return
    stoppedRef.current = true
    const session = sessionRef.current
    const ws = wsRef.current

    if (ws && ws.readyState === WebSocket.OPEN) {
      try {
        ws.send(JSON.stringify({ type: 'flush' }))
      } catch {
        /* ignore */
      }
      // Best-effort: wait briefly for a trailing final.
      await new Promise<void>((resolve) => setTimeout(resolve, 1000))
    }
    await teardown()
    setState('ended')

    if (session && id) {
      try {
        if (kind === 'meeting') await asrService.endForMeeting(id, session.session_id)
        else await asrService.end(id, session.session_id)
      } catch {
        /* best-effort */
      }
    }
    sessionRef.current = null
    setSessionId(null)
  }, [id, kind, teardown])

  // ── Cleanup on unmount ─────────────────────────────────────────────────────
  useEffect(() => {
    return () => {
      stoppedRef.current = true
      void teardown()
    }
  }, [teardown])

  return { state, partial, finals, error, engineMode, sessionId, start, stop }
}

function normaliseError(e: unknown): { code: string; message: string } {
  if (typeof e === 'object' && e !== null && 'response' in e) {
    const r = (e as { response?: { data?: { error?: { code?: string; message?: string } } } })
      .response
    const errObj = r?.data?.error
    if (errObj?.code) return { code: errObj.code, message: errObj.message ?? errObj.code }
  }
  if (e instanceof Error) return { code: 'ASR_FRONTEND_ERROR', message: e.message }
  return { code: 'ASR_FRONTEND_ERROR', message: 'unknown error' }
}
