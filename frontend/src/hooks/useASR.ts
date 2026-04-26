import { useCallback, useEffect, useRef, useState } from 'react'

import { asrService } from '@/services/asrService'
import type {
  ASRFinalEvent,
  ASRHookState,
  ASRServerEvent,
  ASRSessionPayload,
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
  start: () => Promise<void>
  stop: () => Promise<void>
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

export function useASR(id: string | null, kind: ASRSessionKind = 'call'): UseASRResult {
  const [state, setState] = useState<ASRHookState>('idle')
  const [partial, setPartial] = useState('')
  const [finals, setFinals] = useState<ASRFinalEvent[]>([])
  const [error, setError] = useState<{ code: string; message: string } | null>(null)

  const wsRef            = useRef<WebSocket | null>(null)
  const audioCtxRef      = useRef<AudioContext | null>(null)
  const workletNodeRef   = useRef<AudioWorkletNode | null>(null)
  const mediaStreamRef   = useRef<MediaStream | null>(null)
  const sessionRef       = useRef<ASRSessionPayload | null>(null)
  const reconnectsRef    = useRef(0)
  const stoppedRef       = useRef(false)

  // ── Cleanup of audio + WS resources ────────────────────────────────────────
  const teardown = useCallback(async () => {
    if (workletNodeRef.current) {
      workletNodeRef.current.disconnect()
      workletNodeRef.current = null
    }
    if (audioCtxRef.current) {
      try { await audioCtxRef.current.close() } catch { /* ignore */ }
      audioCtxRef.current = null
    }
    if (mediaStreamRef.current) {
      mediaStreamRef.current.getTracks().forEach((t) => t.stop())
      mediaStreamRef.current = null
    }
    if (wsRef.current) {
      try { wsRef.current.close(1000, 'client teardown') } catch { /* ignore */ }
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
      session = kind === 'meeting'
        ? await asrService.requestForMeeting(id, {})
        : await asrService.request(id, {})
      sessionRef.current = session
      logSafe('session issued', { kind, session_id: session.session_id, model_id: session.model_id })
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
        case 'partial':
          setPartial(parsed.text)
          break
        case 'final':
          setPartial('')
          setFinals((cur) => [...cur, parsed])
          break
        case 'error':
          setError({ code: parsed.code, message: parsed.message })
          break
        default:
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
    stoppedRef.current = false
    reconnectsRef.current = 0
    setFinals([])
    setPartial('')
    await openSession()
  }, [openSession])

  const stop = useCallback(async () => {
    stoppedRef.current = true
    const session = sessionRef.current
    const ws = wsRef.current

    if (ws && ws.readyState === WebSocket.OPEN) {
      try { ws.send(JSON.stringify({ type: 'flush' })) } catch { /* ignore */ }
      // Best-effort: wait briefly for a trailing final.
      await new Promise<void>((resolve) => setTimeout(resolve, 1000))
    }
    await teardown()
    setState('ended')

    if (session && id) {
      try {
        if (kind === 'meeting') await asrService.endForMeeting(id, session.session_id)
        else                    await asrService.end(id, session.session_id)
      } catch { /* best-effort */ }
    }
    sessionRef.current = null
  }, [id, kind, teardown])

  // ── Cleanup on unmount ─────────────────────────────────────────────────────
  useEffect(() => {
    return () => { stoppedRef.current = true; void teardown() }
  }, [teardown])

  return { state, partial, finals, error, start, stop }
}

function normaliseError(e: unknown): { code: string; message: string } {
  if (typeof e === 'object' && e !== null && 'response' in e) {
    const r = (e as { response?: { data?: { error?: { code?: string; message?: string } } } }).response
    const errObj = r?.data?.error
    if (errObj?.code) return { code: errObj.code, message: errObj.message ?? errObj.code }
  }
  if (e instanceof Error) return { code: 'ASR_FRONTEND_ERROR', message: e.message }
  return { code: 'ASR_FRONTEND_ERROR', message: 'unknown error' }
}
