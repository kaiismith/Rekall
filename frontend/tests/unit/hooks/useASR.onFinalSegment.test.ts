/**
 * useASR.onFinalSegment unit tests
 *
 * Verifies the persistence callback path added by the transcript-persistence
 * spec (Task 10):
 *   - `onFinalSegment` fires exactly once per `final` event from the WS.
 *   - It NEVER fires for `partial` events.
 *   - The callback receives the full ASRFinalEvent payload (not a coerced
 *     subset), so callers can post the per-word timings + confidence + lang.
 *
 * The hook's audio + AudioWorklet plumbing is mocked out — these tests only
 * exercise the WS message dispatcher branch.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act, waitFor } from '@testing-library/react'

// ── Module mocks ──────────────────────────────────────────────────────────────

vi.mock('@/services/asrService', () => ({
  asrService: {
    request: vi.fn().mockResolvedValue({
      session_id: '11111111-1111-1111-1111-111111111111',
      session_token: 'opaque-token',
      ws_url: 'ws://test/asr',
      expires_at: new Date(Date.now() + 60_000).toISOString(),
      model_id: 'whisper-1',
      sample_rate: 16000,
      frame_format: 'pcm_s16le_mono',
    }),
    requestForMeeting: vi.fn(),
    end: vi.fn().mockResolvedValue({}),
    endForMeeting: vi.fn(),
    buildAudioWorkletUrl: vi.fn().mockReturnValue('/pcm-worklet.js'),
  },
}))

// ── Browser API stubs ────────────────────────────────────────────────────────

const wsInstances: MockWebSocket[] = []

class MockWebSocket {
  static OPEN = 1
  static CLOSED = 3
  readyState = 0
  onopen: (() => void) | null = null
  onmessage: ((e: { data: string }) => void) | null = null
  onclose: ((e: { code: number; reason?: string }) => void) | null = null
  onerror: ((e: Event) => void) | null = null
  send = vi.fn()
  close = vi.fn(() => {
    this.readyState = MockWebSocket.CLOSED
  })

  constructor(public url: string) {
    wsInstances.push(this)
  }

  _msg(payload: Record<string, unknown>) {
    this.onmessage?.({ data: JSON.stringify(payload) })
  }
}

// Minimal AudioContext / Worklet / getUserMedia stubs so the hook's onopen
// branch doesn't throw. We don't drive any audio through them.
class MockAudioContext {
  destination = {}
  audioWorklet = { addModule: vi.fn().mockResolvedValue(undefined) }
  createMediaStreamSource = vi.fn().mockReturnValue({ connect: vi.fn() })
  close = vi.fn().mockResolvedValue(undefined)
}
class MockAudioWorkletNode {
  port = { onmessage: null as ((ev: MessageEvent<ArrayBuffer>) => void) | null }
  connect = vi.fn()
  disconnect = vi.fn()
}

beforeEach(() => {
  wsInstances.length = 0
  // @ts-expect-error — assigning to global for the test environment.
  global.WebSocket = MockWebSocket
  // @ts-expect-error — assigning to global for the test environment.
  global.AudioContext = MockAudioContext
  // @ts-expect-error — assigning to global for the test environment.
  global.AudioWorkletNode = MockAudioWorkletNode
  Object.defineProperty(global.navigator, 'mediaDevices', {
    configurable: true,
    value: {
      getUserMedia: vi.fn().mockResolvedValue({
        getTracks: () => [{ stop: vi.fn() }],
      }),
    },
  })
})

afterEach(() => {
  // NOTE: do NOT call vi.restoreAllMocks() here — it tears down the module-
  // level vi.mock() of @/services/asrService, breaking subsequent tests.
})

// ── Tests ────────────────────────────────────────────────────────────────────

import { useASR } from '@/hooks/useASR'
import type { ASRFinalEvent } from '@/types/asr'

describe('useASR.onFinalSegment', () => {
  it('fires exactly once per final, never for partial, with the full payload', async () => {
    const onFinalSegment = vi.fn<(event: ASRFinalEvent) => void>()
    const { result } = renderHook(() => useASR('call-1', 'call', { onFinalSegment }))

    await act(async () => {
      await result.current.start()
    })

    const ws = wsInstances[0]
    expect(ws).toBeTruthy()

    // Open the WS so the hook moves to 'streaming'.
    act(() => {
      ws._msg({ type: 'ready', session_id: 'sid', model_id: 'whisper-1', sample_rate: 16000 })
    })

    // A partial MUST NOT fire onFinalSegment.
    act(() => {
      ws._msg({
        type: 'partial',
        segment_id: 0,
        text: 'hello',
        start_ms: 0,
        end_ms: 500,
        confidence: 0.5,
      })
    })
    expect(onFinalSegment).not.toHaveBeenCalled()

    // A final MUST fire onFinalSegment exactly once with the full payload.
    const finalPayload = {
      type: 'final',
      segment_id: 0,
      text: 'hello world',
      language: 'en',
      start_ms: 0,
      end_ms: 1500,
      confidence: 0.91,
      words: [
        { w: 'hello', start_ms: 0, end_ms: 700, p: 0.92 },
        { w: 'world', start_ms: 750, end_ms: 1500, p: 0.89 },
      ],
    }
    act(() => {
      ws._msg(finalPayload)
    })
    expect(onFinalSegment).toHaveBeenCalledTimes(1)
    expect(onFinalSegment).toHaveBeenCalledWith(finalPayload)

    // A second final increments the call count, NOT the partial in between.
    act(() => {
      ws._msg({
        type: 'partial',
        segment_id: 1,
        text: 'second',
        start_ms: 1500,
        end_ms: 2000,
        confidence: 0.6,
      })
      ws._msg({
        type: 'final',
        segment_id: 1,
        text: 'second segment',
        language: 'en',
        start_ms: 1500,
        end_ms: 3000,
      })
    })
    expect(onFinalSegment).toHaveBeenCalledTimes(2)
  })

  it('exposes sessionId once the asr token has been issued', async () => {
    const { result } = renderHook(() => useASR('call-1', 'call'))

    expect(result.current.sessionId).toBeNull()

    await act(async () => {
      await result.current.start()
    })

    await waitFor(() => {
      expect(result.current.sessionId).toBe('11111111-1111-1111-1111-111111111111')
    })
  })

  it('does not throw when no callbacks are provided', async () => {
    const { result } = renderHook(() => useASR('call-1', 'call'))

    await act(async () => {
      await result.current.start()
    })
    const ws = wsInstances[0]
    act(() => {
      ws._msg({
        type: 'final',
        segment_id: 0,
        text: 'hello',
        language: 'en',
        start_ms: 0,
        end_ms: 1000,
      })
    })
    // No assertion needed — we're confirming the hook doesn't crash on
    // optional callback access.
    expect(result.current.finals).toHaveLength(1)
  })
})
