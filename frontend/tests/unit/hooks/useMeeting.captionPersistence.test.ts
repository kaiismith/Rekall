/**
 * useMeeting.sendCaptionFinal unit tests
 *
 * Verifies the persistence-shape relay path added by the transcript-
 * persistence spec (Task 10.5):
 *   - sendCaptionFinal dispatches a `caption_chunk` WS message that BOTH
 *     carries the legacy fields (caption_kind/caption_text/caption_segment_id)
 *     AND the new persistence-shape fields (session_id, segment_index,
 *     start_ms, end_ms, language, confidence, words).
 *   - sendCaptionChunk (the legacy partial path) does NOT include the
 *     persistence fields — backend hub treats the absence as "relay only,
 *     don't persist", which is what we want for partials.
 *
 * The WebRTC + AudioContext stack is mocked just enough that the hook can
 * reach 'joined' state. We don't drive any media — only WS message dispatch.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'

// ── Module mocks ──────────────────────────────────────────────────────────────

vi.mock('@/store/authStore', () => ({
  useAuthStore: () => ({
    accessToken: 'fake-token',
    user: {
      id: 'user-1',
      email: 'u1@example.com',
      full_name: 'User One',
      role: 'member',
      email_verified: true,
      created_at: '2026-01-01T00:00:00Z',
    },
  }),
}))

const mockMeeting = {
  id: 'meeting-1',
  code: 'test-code',
  title: 'Test Meeting',
  type: 'open',
  host_id: 'user-1',
  status: 'active',
  max_participants: 10,
  transcription_enabled: true,
  join_url: 'http://localhost/meeting/test-code',
  created_at: '2026-01-01T00:00:00Z',
}

vi.mock('@/services/meetingService', () => ({
  meetingService: {
    getByCode: vi.fn().mockResolvedValue({ data: mockMeeting }),
    requestWsTicket: vi.fn().mockResolvedValue({
      ticket: 'opaque-xyz',
      wsUrl: '/api/v1/meetings/test-code/ws?ticket=opaque-xyz',
      expiresAt: Date.now() + 60_000,
    }),
    buildAbsoluteWsUrl: vi
      .fn()
      .mockReturnValue('ws://localhost/api/v1/meetings/test-code/ws?ticket=opaque-xyz'),
    end: vi.fn().mockResolvedValue({}),
    listMessages: vi.fn().mockResolvedValue({ messages: [], has_more: false }),
  },
}))

// VadDetector + VirtualBackgroundPipeline produce no-op shells so the hook's
// init effects don't blow up on jsdom.
vi.mock('@/utils/vadDetector', () => ({
  VadDetector: class {
    start = vi.fn()
    stop = vi.fn()
    onSpeakingChange = vi.fn()
  },
}))
vi.mock('@/utils/virtualBackgroundPipeline', () => {
  class MockVBP {
    static isSupported() {
      return false
    }
    start = vi.fn()
    stop = vi.fn()
  }
  return { VirtualBackgroundPipeline: MockVBP }
})

// ── Browser API stubs ────────────────────────────────────────────────────────

const wsInstances: MockWebSocket[] = []

class MockWebSocket {
  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3
  readyState: number = MockWebSocket.CONNECTING
  onopen: (() => void) | null = null
  onmessage: ((e: { data: string }) => void) | null = null
  onclose: ((e: { code: number }) => void) | null = null
  onerror: ((e: Event) => void) | null = null
  send = vi.fn()
  close = vi.fn(() => {
    this.readyState = MockWebSocket.CLOSED
  })
  constructor(public url: string) {
    wsInstances.push(this)
  }
  _open() {
    this.readyState = MockWebSocket.OPEN
    this.onopen?.()
  }
  _msg(payload: Record<string, unknown>) {
    this.onmessage?.({ data: JSON.stringify(payload) })
  }
  /** Decode the most recent send() call as JSON for inspection. */
  lastSent(): Record<string, unknown> | null {
    if (this.send.mock.calls.length === 0) return null
    const raw = this.send.mock.calls[this.send.mock.calls.length - 1][0]
    return JSON.parse(raw as string) as Record<string, unknown>
  }
  /** All sends so far as parsed JSON, in order. */
  allSent(): Record<string, unknown>[] {
    return this.send.mock.calls.map((c) => JSON.parse(c[0] as string) as Record<string, unknown>)
  }
}

class MockRTCPeerConnection {
  iceConnectionState = 'new'
  signalingState = 'stable'
  ontrack: ((ev: RTCTrackEvent) => void) | null = null
  onicecandidate: ((ev: RTCPeerConnectionIceEvent) => void) | null = null
  onconnectionstatechange: (() => void) | null = null
  oniceconnectionstatechange: (() => void) | null = null
  onnegotiationneeded: (() => void) | null = null
  addTrack = vi.fn()
  addTransceiver = vi.fn()
  close = vi.fn()
  getSenders = vi.fn(() => [])
  createOffer = vi.fn().mockResolvedValue({ type: 'offer', sdp: 'offer-sdp' })
  createAnswer = vi.fn().mockResolvedValue({ type: 'answer', sdp: 'answer-sdp' })
  setLocalDescription = vi.fn().mockResolvedValue(undefined)
  setRemoteDescription = vi.fn().mockResolvedValue(undefined)
  addIceCandidate = vi.fn().mockResolvedValue(undefined)
}

class MockAudioCtx {
  createAnalyser = vi.fn(() => ({
    fftSize: 512,
    smoothingTimeConstant: 0,
    getFloatTimeDomainData: vi.fn(),
    connect: vi.fn(),
  }))
  createMediaStreamSource = vi.fn(() => ({ connect: vi.fn(), disconnect: vi.fn() }))
  close = vi.fn()
}

const mockAudioTrack = {
  kind: 'audio',
  enabled: true,
  stop: vi.fn(),
} as unknown as MediaStreamTrack
const mockVideoTrack = {
  kind: 'video',
  enabled: true,
  stop: vi.fn(),
} as unknown as MediaStreamTrack
const mockStream = {
  getAudioTracks: () => [mockAudioTrack],
  getVideoTracks: () => [mockVideoTrack],
  getTracks: () => [mockAudioTrack, mockVideoTrack],
} as unknown as MediaStream

beforeEach(() => {
  wsInstances.length = 0
  vi.stubGlobal('WebSocket', MockWebSocket)
  vi.stubGlobal('RTCPeerConnection', MockRTCPeerConnection)
  vi.stubGlobal(
    'AudioContext',
    vi.fn(() => new MockAudioCtx()),
  )
  vi.stubGlobal(
    'MediaStream',
    class {
      private _tracks: MediaStreamTrack[] = []
      addTrack(t: MediaStreamTrack) {
        this._tracks.push(t)
      }
      getTracks() {
        return this._tracks
      }
    },
  )
  Object.defineProperty(navigator, 'mediaDevices', {
    value: {
      getUserMedia: vi.fn().mockResolvedValue(mockStream),
      getDisplayMedia: vi.fn().mockResolvedValue(mockStream),
    },
    configurable: true,
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

// ── Helper: render the hook + drive it to a state where send() is wired ─────

async function renderJoinedMeeting() {
  const { useMeeting } = await import('@/hooks/useMeeting')
  const hook = renderHook(() => useMeeting({ code: 'test-code' }))

  // Wait for getUserMedia → localStream.
  await act(async () => {
    await vi.waitFor(() => hook.result.current.localStream != null)
  })

  // Leave the device-check screen so the WS connect effect fires.
  act(() => {
    hook.result.current.joinNow()
  })

  // Wait for the WS to be created.
  await act(async () => {
    await vi.waitFor(() => wsInstances.length > 0)
  })

  // Open the WS so send() actually transmits.
  act(() => {
    wsInstances[0]._open()
  })

  return { hook, ws: wsInstances[0] }
}

// ── Tests ────────────────────────────────────────────────────────────────────

describe('useMeeting caption persistence path', () => {
  it('sendCaptionFinal dispatches the persistence-shape payload', async () => {
    const { hook, ws } = await renderJoinedMeeting()

    // Sanity: function is exported.
    expect(typeof hook.result.current.sendCaptionFinal).toBe('function')

    const before = ws.send.mock.calls.length

    act(() => {
      hook.result.current.sendCaptionFinal(
        {
          type: 'final',
          segment_id: 7,
          text: 'persistence please',
          language: 'en',
          start_ms: 12340,
          end_ms: 13880,
          confidence: 0.91,
          words: [
            { w: 'persistence', start_ms: 12340, end_ms: 13000, p: 0.93 },
            { w: 'please', start_ms: 13050, end_ms: 13880, p: 0.89 },
          ],
        },
        '11111111-1111-1111-1111-111111111111',
      )
    })

    // Find the caption_chunk send (other messages may be in flight after join).
    const sent = ws.allSent().slice(before)
    const chunk = sent.find((m) => m.type === 'caption_chunk')
    expect(chunk).toBeTruthy()

    // Legacy fields preserved.
    expect(chunk!.caption_kind).toBe('final')
    expect(chunk!.caption_text).toBe('persistence please')
    expect(chunk!.caption_segment_id).toBe('7')

    // Persistence-shape fields populated.
    expect(chunk!.session_id).toBe('11111111-1111-1111-1111-111111111111')
    expect(chunk!.segment_index).toBe(7)
    expect(chunk!.start_ms).toBe(12340)
    expect(chunk!.end_ms).toBe(13880)
    expect(chunk!.language).toBe('en')
    expect(chunk!.confidence).toBeCloseTo(0.91)
    expect(chunk!.words).toHaveLength(2)
  })

  it('sendCaptionChunk (legacy partial path) does NOT include persistence fields', async () => {
    const { hook, ws } = await renderJoinedMeeting()

    const before = ws.send.mock.calls.length

    act(() => {
      hook.result.current.sendCaptionChunk('partial', 'hello', 'seg-0')
    })

    const sent = ws.allSent().slice(before)
    const chunk = sent.find((m) => m.type === 'caption_chunk')
    expect(chunk).toBeTruthy()
    expect(chunk!.caption_kind).toBe('partial')
    expect(chunk!.caption_text).toBe('hello')
    expect(chunk!.caption_segment_id).toBe('seg-0')

    // None of the persistence-shape fields must be present — backend treats
    // their absence as "relay only, don't persist", which is correct for
    // partials (we never persist mid-utterance state).
    expect(chunk!.session_id).toBeUndefined()
    expect(chunk!.segment_index).toBeUndefined()
    expect(chunk!.start_ms).toBeUndefined()
    expect(chunk!.end_ms).toBeUndefined()
    expect(chunk!.words).toBeUndefined()
  })
})
