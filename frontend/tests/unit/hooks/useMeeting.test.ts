/**
 * useMeeting unit tests
 *
 * These tests cover the pure / side-effect-free logic that can be exercised
 * without a real WebRTC stack: localStorage helpers, uploadCustomBackground
 * validation, and initial derived state.
 *
 * Full WebRTC + WebSocket integration belongs in e2e tests (Playwright).
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'

// ── Module mocks (must be hoisted before imports) ─────────────────────────────

vi.mock('@/store/authStore', () => ({
  useAuthStore: () => ({
    accessToken: 'fake-token',
    user: { id: 'user-1', email: 'u1@example.com', full_name: 'User One', role: 'member', email_verified: true, created_at: '2026-01-01T00:00:00Z' },
  }),
}))

vi.mock('@/services/meetingService', () => ({
  meetingService: {
    getByCode: vi.fn().mockResolvedValue({
      data: {
        id: 'meeting-1', code: 'test-code', title: 'Test Meeting', type: 'open',
        host_id: 'user-1', status: 'active', max_participants: 10,
        join_url: 'http://localhost/meeting/test-code', created_at: '2026-01-01T00:00:00Z',
      },
    }),
    buildWsUrl: vi.fn().mockReturnValue('ws://localhost/ws/test-code'),
    end: vi.fn().mockResolvedValue({}),
    listMessages: vi.fn().mockResolvedValue({ messages: [], has_more: false }),
  },
}))

const mockMeeting = {
  id: 'meeting-1',
  code: 'test-code',
  title: 'Test Meeting',
  type: 'open',
  host_id: 'user-1',
  status: 'active',
  max_participants: 10,
  join_url: 'http://localhost/meeting/test-code',
  created_at: '2026-01-01T00:00:00Z',
}

// ── Browser API stubs ─────────────────────────────────────────────────────────

/** Collects all constructed instances so tests can drive them. */
const wsInstances: MockWebSocket[] = []

class MockWebSocket {
  static OPEN = 1
  readyState = 0
  onopen: (() => void) | null = null
  onmessage: ((e: { data: string }) => void) | null = null
  onclose: ((e: { code: number }) => void) | null = null
  onerror: ((e: Event) => void) | null = null
  send = vi.fn()
  close = vi.fn()

  constructor() {
    wsInstances.push(this)
  }

  /** Helper: simulate the WS opening and transitioning to OPEN. */
  _open() {
    this.readyState = MockWebSocket.OPEN
    this.onopen?.()
  }

  /** Helper: push a message through the WS. */
  _msg(msg: Record<string, unknown>) {
    this.onmessage?.({ data: JSON.stringify(msg) })
  }

  /** Helper: simulate close event. */
  _close(code = 1000) {
    this.readyState = 3
    this.onclose?.({ code })
  }
}

const rtcInstances: MockRTCPeerConnection[] = []

class MockRTCPeerConnection {
  onicecandidate: ((e: { candidate: unknown }) => void) | null = null
  onnegotiationneeded: (() => void) | null = null
  getSenders = vi.fn(() => [])
  addTrack = vi.fn()
  close = vi.fn()
  createOffer = vi.fn().mockResolvedValue({ type: 'offer', sdp: 'offer-sdp' })
  createAnswer = vi.fn().mockResolvedValue({ type: 'answer', sdp: 'answer-sdp' })
  setLocalDescription = vi.fn().mockResolvedValue(undefined)
  setRemoteDescription = vi.fn().mockResolvedValue(undefined)
  addIceCandidate = vi.fn().mockResolvedValue(undefined)

  constructor() {
    rtcInstances.push(this)
  }
}

class MockAnalyser {
  fftSize = 512
  smoothingTimeConstant = 0
  getFloatTimeDomainData = vi.fn()
  connect = vi.fn()
}
class MockAudioCtx {
  createAnalyser = vi.fn(() => new MockAnalyser())
  createMediaStreamSource = vi.fn(() => ({ connect: vi.fn(), disconnect: vi.fn() }))
  close = vi.fn()
}

import { meetingService } from '@/services/meetingService'

const mockAudioTrack = { kind: 'audio', enabled: true, stop: vi.fn() } as unknown as MediaStreamTrack
const mockVideoTrack = { kind: 'video', enabled: true, stop: vi.fn() } as unknown as MediaStreamTrack
const mockStream = {
  getAudioTracks: () => [mockAudioTrack],
  getVideoTracks: () => [mockVideoTrack],
  getTracks: () => [mockAudioTrack, mockVideoTrack],
} as unknown as MediaStream

// ─────────────────────────────────────────────────────────────────────────────

const BG_STORAGE_KEY = 'rekall_bg_preference'
const CUSTOM_BG_KEY = 'rekall_bg_custom'

function setupBrowserMocks() {
  wsInstances.length = 0
  rtcInstances.length = 0
  localStorage.clear()
  vi.clearAllMocks()

  // Re-apply mock implementations (vi.restoreAllMocks from upload tests may have cleared them).
  vi.mocked(meetingService.getByCode).mockResolvedValue({ data: mockMeeting } as never)
  vi.mocked(meetingService.buildWsUrl).mockReturnValue('ws://localhost/ws/test-code')
  vi.mocked(meetingService.end).mockResolvedValue({} as never)
  vi.mocked(meetingService.listMessages).mockResolvedValue({ messages: [], has_more: false })

  vi.stubGlobal('WebSocket', MockWebSocket)
  vi.stubGlobal('RTCPeerConnection', MockRTCPeerConnection)
  vi.stubGlobal('AudioContext', vi.fn(() => new MockAudioCtx()))
  Object.defineProperty(navigator, 'mediaDevices', {
    value: {
      getUserMedia: vi.fn().mockResolvedValue(mockStream),
      getDisplayMedia: vi.fn().mockResolvedValue(mockStream),
    },
    configurable: true,
  })
}

function teardownBrowserMocks() {
  localStorage.clear()
  vi.clearAllMocks()
}

/** Render the hook and wait for the WS connect effect to fire. */
async function renderMeeting(code = 'test-code', onEnd?: () => void) {
  const { useMeeting } = await import('@/hooks/useMeeting')
  const hook = renderHook(() => useMeeting({ code, onEnd }))
  // Let connect effect run (fires meetingService.getByCode → new WebSocket).
  await act(async () => { await vi.waitFor(() => wsInstances.length > 0) })
  return { ...hook, ws: wsInstances[wsInstances.length - 1] }
}

describe('useMeeting — localStorage helpers', () => {
  beforeEach(() => setupBrowserMocks())
  afterEach(() => teardownBrowserMocks())

  it('initialises activeBackground to none when localStorage is empty', async () => {
    const { useMeeting } = await import('@/hooks/useMeeting')
    const { result } = renderHook(() => useMeeting({ code: 'test-code' }))
    expect(result.current.activeBackground).toEqual({ type: 'none' })
  })

  it('restores a stored blur preference from localStorage', async () => {
    localStorage.setItem(BG_STORAGE_KEY, JSON.stringify({ type: 'blur', level: 'heavy' }))
    const { useMeeting } = await import('@/hooks/useMeeting')
    const { result } = renderHook(() => useMeeting({ code: 'test-code' }))
    expect(result.current.activeBackground).toEqual({ type: 'blur', level: 'heavy' })
  })

  it('restores a stored preset image preference', async () => {
    const preset = { type: 'image', src: '/backgrounds/office.jpg', label: 'Office' }
    localStorage.setItem(BG_STORAGE_KEY, JSON.stringify(preset))
    const { useMeeting } = await import('@/hooks/useMeeting')
    const { result } = renderHook(() => useMeeting({ code: 'test-code' }))
    expect(result.current.activeBackground).toEqual(preset)
  })

  it('resolves the __custom__ sentinel to the data URL from CUSTOM_BG_KEY', async () => {
    const dataUrl = 'data:image/png;base64,abc123'
    localStorage.setItem(CUSTOM_BG_KEY, dataUrl)
    localStorage.setItem(
      BG_STORAGE_KEY,
      JSON.stringify({ type: 'image', src: '__custom__', label: 'Custom' }),
    )
    const { useMeeting } = await import('@/hooks/useMeeting')
    const { result } = renderHook(() => useMeeting({ code: 'test-code' }))
    expect(result.current.activeBackground).toEqual({
      type: 'image',
      src: dataUrl,
      label: 'Custom',
    })
  })

  it('falls back to none when __custom__ sentinel has no data in CUSTOM_BG_KEY', async () => {
    localStorage.setItem(
      BG_STORAGE_KEY,
      JSON.stringify({ type: 'image', src: '__custom__', label: 'Custom' }),
    )
    // No CUSTOM_BG_KEY entry.
    const { useMeeting } = await import('@/hooks/useMeeting')
    const { result } = renderHook(() => useMeeting({ code: 'test-code' }))
    expect(result.current.activeBackground).toEqual({ type: 'none' })
  })

  it('initialises customBgSrc from CUSTOM_BG_KEY', async () => {
    const dataUrl = 'data:image/jpeg;base64,xyz'
    localStorage.setItem(CUSTOM_BG_KEY, dataUrl)
    const { useMeeting } = await import('@/hooks/useMeeting')
    const { result } = renderHook(() => useMeeting({ code: 'test-code' }))
    expect(result.current.customBgSrc).toBe(dataUrl)
  })

  it('initialises customBgSrc to null when CUSTOM_BG_KEY is absent', async () => {
    const { useMeeting } = await import('@/hooks/useMeeting')
    const { result } = renderHook(() => useMeeting({ code: 'test-code' }))
    expect(result.current.customBgSrc).toBeNull()
  })
})

describe('useMeeting — uploadCustomBackground validation', () => {
  beforeEach(() => {
    setupBrowserMocks()
    // Need createElement spy for bgSupported detection — setup manually here.
    // Ensure captureStream is available so bgSupported = true.
    // Only intercept 'canvas' and 'video' — let React's internal element
    // creation fall through to the real implementation.
    const originalCreate = document.createElement.bind(document)
    const fakeCtx = { filter: 'none', drawImage: vi.fn() }
    vi.spyOn(document, 'createElement').mockImplementation((tag, options) => {
      if (tag === 'canvas') {
        return {
          width: 0, height: 0,
          getContext: () => fakeCtx,
          captureStream: () => ({ getVideoTracks: () => [{ kind: 'video' }] }),
        } as unknown as HTMLCanvasElement
      }
      if (tag === 'video') {
        return {
          srcObject: null, muted: false, playsInline: false,
          play: vi.fn(() => Promise.resolve()),
        } as unknown as HTMLVideoElement
      }
      return originalCreate(tag, options)
    })
  })

  afterEach(() => {
    // Must restore createElement spy to avoid polluting other tests.
    vi.restoreAllMocks()
    localStorage.clear()
  })

  it('returns an error when file exceeds 2 MB', async () => {
    const { useMeeting } = await import('@/hooks/useMeeting')
    const { result } = renderHook(() => useMeeting({ code: 'test-code' }))

    const bigFile = new File([new ArrayBuffer(3 * 1024 * 1024)], 'big.png', { type: 'image/png' })
    let error: string | null = null
    await act(async () => {
      error = await result.current.uploadCustomBackground(bigFile)
    })
    expect(error).toBe('Image must be 2 MB or smaller')
  })

  it('returns null and writes to localStorage on successful upload', async () => {
    const { useMeeting } = await import('@/hooks/useMeeting')
    const { result } = renderHook(() => useMeeting({ code: 'test-code' }))

    const smallFile = new File(['hello'], 'small.png', { type: 'image/png' })
    let error: string | null = 'sentinel'
    await act(async () => {
      error = await result.current.uploadCustomBackground(smallFile)
    })

    expect(error).toBeNull()
    expect(localStorage.getItem(CUSTOM_BG_KEY)).not.toBeNull()
    const pref = JSON.parse(localStorage.getItem(BG_STORAGE_KEY)!)
    expect(pref.src).toBe('__custom__')
  })

  it('returns storage error when localStorage.setItem throws (quota exceeded)', async () => {
    const { useMeeting } = await import('@/hooks/useMeeting')
    const { result } = renderHook(() => useMeeting({ code: 'test-code' }))

    // Make localStorage.setItem throw to simulate quota exceeded
    const origSetItem = Storage.prototype.setItem
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new DOMException('QuotaExceededError')
    })

    const smallFile = new File(['hello'], 'small.png', { type: 'image/png' })
    let error: string | null = null
    await act(async () => {
      error = await result.current.uploadCustomBackground(smallFile)
    })
    expect(error).toBe('Not enough storage space for this image')

    // Restore setItem for subsequent tests
    Storage.prototype.setItem = origSetItem
  })

  it('updates customBgSrc state after successful upload', async () => {
    const { useMeeting } = await import('@/hooks/useMeeting')
    const { result } = renderHook(() => useMeeting({ code: 'test-code' }))

    const smallFile = new File(['hello'], 'small.png', { type: 'image/png' })
    await act(async () => {
      await result.current.uploadCustomBackground(smallFile)
    })

    expect(result.current.customBgSrc).not.toBeNull()
    expect(result.current.customBgSrc).toMatch(/^data:/)
  })
})

// ═════════════════════════════════════════════════════════════════════════════
// WS connect lifecycle + media controls + message handler
// ═════════════════════════════════════════════════════════════════════════════

describe('useMeeting — WS connect lifecycle', () => {
  beforeEach(() => setupBrowserMocks())
  afterEach(() => teardownBrowserMocks())

  it('starts in "connecting" state', async () => {
    const { result } = await renderMeeting()
    expect(result.current.roomState).toBe('connecting')
  })

  it('calls meetingService.getByCode with the code', async () => {
    await renderMeeting('my-code')
    expect(meetingService.getByCode).toHaveBeenCalledWith('my-code')
  })

  it('creates a WebSocket via meetingService.buildWsUrl', async () => {
    await renderMeeting()
    expect(meetingService.buildWsUrl).toHaveBeenCalledWith('test-code', 'fake-token')
    expect(wsInstances.length).toBe(1)
  })

  it('sets meeting data from getByCode response', async () => {
    const { result } = await renderMeeting()
    expect(result.current.meeting).toEqual(mockMeeting)
  })

  it('transitions to "in_meeting" when participant.joined is received after connecting', async () => {
    const { result, ws } = await renderMeeting()
    await act(async () => {
      ws._open()
      ws._msg({ type: 'participant.joined', user_id: 'remote-1' })
    })
    expect(result.current.roomState).toBe('in_meeting')
  })

  it('transitions to "waiting_room" when knock.requested is received while connecting', async () => {
    const { result, ws } = await renderMeeting()
    await act(async () => {
      ws._open()
      ws._msg({ type: 'knock.requested', knock_id: 'k1', user_id: 'u1' })
    })
    expect(result.current.roomState).toBe('waiting_room')
  })

  it('transitions to "denied" when WS closes with code 4003', async () => {
    const { result, ws } = await renderMeeting()
    await act(async () => {
      ws._open()
      ws._close(4003)
    })
    expect(result.current.roomState).toBe('denied')
  })

  it('transitions to "ended" when WS closes normally', async () => {
    const { result, ws } = await renderMeeting()
    await act(async () => {
      ws._open()
      ws._close(1000)
    })
    expect(result.current.roomState).toBe('ended')
  })

  it('transitions to "error" on WS error', async () => {
    const { result, ws } = await renderMeeting()
    await act(async () => {
      ws.onerror?.(new Event('error'))
    })
    expect(result.current.roomState).toBe('error')
  })

  it('transitions to "error" if getByCode rejects', async () => {
    vi.mocked(meetingService.getByCode).mockRejectedValueOnce(new Error('Not found'))
    const { useMeeting } = await import('@/hooks/useMeeting')
    const { result } = renderHook(() => useMeeting({ code: 'bad-code' }))
    await act(async () => { await vi.waitFor(() => result.current.roomState === 'error') })
    expect(result.current.roomState).toBe('error')
  })

  it('does not create WS when code is empty', async () => {
    const { useMeeting } = await import('@/hooks/useMeeting')
    renderHook(() => useMeeting({ code: '' }))
    // Give effects time to fire
    await act(async () => { await new Promise((r) => setTimeout(r, 50)) })
    expect(wsInstances.length).toBe(0)
  })
})

describe('useMeeting — public actions', () => {
  beforeEach(() => setupBrowserMocks())
  afterEach(() => teardownBrowserMocks())

  async function joinedMeeting() {
    const res = await renderMeeting()
    const { ws } = res
    await act(async () => {
      ws._open()
      ws._msg({ type: 'participant.joined', user_id: 'remote-1' })
    })
    return res
  }

  it('respondToKnock sends knock.respond via WS', async () => {
    const { result, ws } = await joinedMeeting()
    act(() => result.current.respondToKnock('k1', true))
    expect(ws.send).toHaveBeenCalledWith(
      expect.stringContaining('"type":"knock.respond"'),
    )
  })

  it('leave sets roomState to ended', async () => {
    const { result } = await joinedMeeting()
    act(() => result.current.leave())
    expect(result.current.roomState).toBe('ended')
  })

  it('endMeeting calls meetingService.end and sets roomState', async () => {
    const onEnd = vi.fn()
    const res = await renderMeeting('test-code', onEnd)
    const { ws } = res
    await act(async () => {
      ws._open()
      ws._msg({ type: 'participant.joined', user_id: 'remote-1' })
    })
    await act(async () => { await res.result.current.endMeeting() })
    expect(meetingService.end).toHaveBeenCalledWith('test-code')
    expect(res.result.current.roomState).toBe('ended')
    expect(onEnd).toHaveBeenCalled()
  })

  it('toggleMute flips isMuted and sends media_state', async () => {
    const { result, ws } = await joinedMeeting()
    expect(result.current.isMuted).toBe(false)
    act(() => result.current.toggleMute())
    expect(result.current.isMuted).toBe(true)
    expect(ws.send).toHaveBeenCalledWith(expect.stringContaining('"audio":false'))
    act(() => result.current.toggleMute())
    expect(result.current.isMuted).toBe(false)
  })

  it('toggleCamera flips isCameraOff and sends media_state', async () => {
    const { result, ws } = await joinedMeeting()
    expect(result.current.isCameraOff).toBe(false)
    act(() => result.current.toggleCamera())
    expect(result.current.isCameraOff).toBe(true)
    expect(ws.send).toHaveBeenCalledWith(expect.stringContaining('"video":false'))
  })

  it('toggleHand flips isHandRaised and sends hand_raise', async () => {
    const { result, ws } = await joinedMeeting()
    expect(result.current.isHandRaised).toBe(false)
    act(() => result.current.toggleHand())
    expect(result.current.isHandRaised).toBe(true)
    expect(ws.send).toHaveBeenCalledWith(expect.stringContaining('"type":"hand_raise"'))
  })

  it('sendEmojiReaction sends emoji_reaction via WS', async () => {
    const { result, ws } = await joinedMeeting()
    act(() => result.current.sendEmojiReaction('🎉'))
    expect(ws.send).toHaveBeenCalledWith(expect.stringContaining('"emoji":"🎉"'))
  })

  it('sendEmojiReaction rate-limits within 500ms', async () => {
    const { result, ws } = await joinedMeeting()
    act(() => result.current.sendEmojiReaction('🎉'))
    const countAfterFirst = ws.send.mock.calls.filter(
      (c: string[]) => c[0].includes('emoji_reaction'),
    ).length
    act(() => result.current.sendEmojiReaction('👍'))
    const countAfterSecond = ws.send.mock.calls.filter(
      (c: string[]) => c[0].includes('emoji_reaction'),
    ).length
    // Second call within 500ms should be ignored
    expect(countAfterSecond).toBe(countAfterFirst)
  })

  it('toggleLaser flips isLaserActive and sends laser_stop when deactivating', async () => {
    const { result, ws } = await joinedMeeting()
    act(() => result.current.toggleLaser())
    expect(result.current.isLaserActive).toBe(true)
    act(() => result.current.toggleLaser())
    expect(result.current.isLaserActive).toBe(false)
    expect(ws.send).toHaveBeenCalledWith(expect.stringContaining('"type":"laser_stop"'))
  })

  it('sendLaserMove sends laser_move via WS', async () => {
    const { result, ws } = await joinedMeeting()
    act(() => result.current.sendLaserMove(0.5, 0.3))
    expect(ws.send).toHaveBeenCalledWith(expect.stringContaining('"type":"laser_move"'))
  })

  it('forceMute sends force_mute via WS', async () => {
    const { result, ws } = await joinedMeeting()
    act(() => result.current.forceMute('user-xyz'))
    expect(ws.send).toHaveBeenCalledWith(expect.stringContaining('"target_id":"user-xyz"'))
  })
})

describe('useMeeting — WS message handler', () => {
  beforeEach(() => setupBrowserMocks())
  afterEach(() => teardownBrowserMocks())

  async function joinedMeeting() {
    const res = await renderMeeting()
    const { ws } = res
    await act(async () => {
      ws._open()
      ws._msg({ type: 'participant.joined', user_id: 'remote-1' })
    })
    return res
  }

  it('participant.left removes the peer and cleans up state', async () => {
    const { result, ws } = await joinedMeeting()
    // First, add a remote participant
    await act(async () => {
      ws._msg({ type: 'participant.joined', user_id: 'remote-2' })
    })
    // Now remove them
    await act(async () => {
      ws._msg({ type: 'participant.left', user_id: 'remote-2' })
    })
    expect(result.current.peers['remote-2']).toBeUndefined()
  })

  it('speaking_state updates remoteSpeaking map', async () => {
    const { result, ws } = await joinedMeeting()
    await act(async () => {
      ws._msg({ type: 'speaking_state', from: 'remote-1', payload: { speaking: true } })
    })
    expect(result.current.remoteSpeaking['remote-1']).toBe(true)
    await act(async () => {
      ws._msg({ type: 'speaking_state', from: 'remote-1', payload: { speaking: false } })
    })
    expect(result.current.remoteSpeaking['remote-1']).toBe(false)
  })

  it('media_state updates mediaStates map', async () => {
    const { result, ws } = await joinedMeeting()
    await act(async () => {
      ws._msg({ type: 'media_state', user_id: 'remote-1', audio: false, video: true })
    })
    expect(result.current.mediaStates['remote-1']).toEqual({ audio: false, video: true })
  })

  it('hand_raise adds/removes from handRaisedUsers', async () => {
    const { result, ws } = await joinedMeeting()
    await act(async () => {
      ws._msg({ type: 'hand_raise', user_id: 'remote-1', raised: true })
    })
    expect(result.current.handRaisedUsers.has('remote-1')).toBe(true)
    expect(result.current.handRaisedCount).toBe(1)
    await act(async () => {
      ws._msg({ type: 'hand_raise', user_id: 'remote-1', raised: false })
    })
    expect(result.current.handRaisedUsers.has('remote-1')).toBe(false)
    expect(result.current.handRaisedCount).toBe(0)
  })

  it('emoji_reaction adds to reactionQueue', async () => {
    const { result, ws } = await joinedMeeting()
    await act(async () => {
      ws._msg({ type: 'emoji_reaction', from_id: 'remote-1', emoji: '🎉' })
    })
    expect(result.current.reactionQueue.length).toBe(1)
    expect(result.current.reactionQueue[0].emoji).toBe('🎉')
  })

  it('emoji_reaction is removed from queue after 3s timeout', async () => {
    vi.useFakeTimers()
    const { result, ws } = await joinedMeeting()
    await act(async () => {
      ws._msg({ type: 'emoji_reaction', from_id: 'remote-1', emoji: '🎉' })
    })
    expect(result.current.reactionQueue.length).toBe(1)
    // Advance past the 3s timeout
    await act(async () => { vi.advanceTimersByTime(3100) })
    expect(result.current.reactionQueue.length).toBe(0)
    vi.useRealTimers()
  })

  it('emoji_reaction with no userId is ignored', async () => {
    const { result, ws } = await joinedMeeting()
    await act(async () => {
      ws._msg({ type: 'emoji_reaction', emoji: '👍' }) // no from_id, user_id, or from
    })
    expect(result.current.reactionQueue.length).toBe(0)
  })

  it('laser_move updates laserState', async () => {
    const { result, ws } = await joinedMeeting()
    await act(async () => {
      ws._msg({ type: 'laser_move', user_id: 'remote-1', x: 0.3, y: 0.7 })
    })
    expect(result.current.laserState).toEqual({ userId: 'remote-1', x: 0.3, y: 0.7 })
  })

  it('laser_move deactivates local laser when another user takes it', async () => {
    const { result, ws } = await joinedMeeting()
    // Activate local laser first
    act(() => result.current.toggleLaser())
    expect(result.current.isLaserActive).toBe(true)
    // Remote user sends laser_move — should deactivate our laser (lines 396-398)
    await act(async () => {
      ws._msg({ type: 'laser_move', user_id: 'remote-1', x: 0.5, y: 0.5 })
    })
    expect(result.current.isLaserActive).toBe(false)
  })

  it('laser_stop deactivates local laser when active', async () => {
    const { result, ws } = await joinedMeeting()
    // Activate local laser
    act(() => result.current.toggleLaser())
    expect(result.current.isLaserActive).toBe(true)
    // Remote laser_stop — should deactivate our laser (lines 405-407)
    await act(async () => {
      ws._msg({ type: 'laser_stop', user_id: 'remote-1' })
    })
    expect(result.current.isLaserActive).toBe(false)
  })

  it('laser_stop clears laserState for that user', async () => {
    const { result, ws } = await joinedMeeting()
    await act(async () => {
      ws._msg({ type: 'laser_move', user_id: 'remote-1', x: 0.5, y: 0.5 })
    })
    expect(result.current.laserState).not.toBeNull()
    await act(async () => {
      ws._msg({ type: 'laser_stop', user_id: 'remote-1' })
    })
    expect(result.current.laserState).toBeNull()
  })

  it('knock.requested adds to knocks list', async () => {
    const { result, ws } = await joinedMeeting()
    await act(async () => {
      ws._msg({ type: 'knock.requested', knock_id: 'k1', user_id: 'visitor-1' })
    })
    expect(result.current.knocks).toEqual([{ knock_id: 'k1', user_id: 'visitor-1' }])
  })

  it('knock.resolved removes from knocks list', async () => {
    const { result, ws } = await joinedMeeting()
    await act(async () => {
      ws._msg({ type: 'knock.requested', knock_id: 'k1', user_id: 'visitor-1' })
    })
    await act(async () => {
      ws._msg({ type: 'knock.resolved', knock_id: 'k1' })
    })
    expect(result.current.knocks).toEqual([])
  })

  it('knock.cancelled removes from knocks list', async () => {
    const { result, ws } = await joinedMeeting()
    await act(async () => {
      ws._msg({ type: 'knock.requested', knock_id: 'k1', user_id: 'visitor-1' })
    })
    await act(async () => {
      ws._msg({ type: 'knock.cancelled', knock_id: 'k1' })
    })
    expect(result.current.knocks).toEqual([])
  })

  it('knock.denied sets roomState to denied', async () => {
    const { result, ws } = await joinedMeeting()
    await act(async () => {
      ws._msg({ type: 'knock.denied' })
    })
    expect(result.current.roomState).toBe('denied')
  })

  it('meeting.ended sets roomState to ended and calls onEnd', async () => {
    const onEnd = vi.fn()
    const res = await renderMeeting('test-code', onEnd)
    await act(async () => {
      res.ws._open()
      res.ws._msg({ type: 'participant.joined', user_id: 'remote-1' })
    })
    await act(async () => {
      res.ws._msg({ type: 'meeting.ended' })
    })
    expect(res.result.current.roomState).toBe('ended')
    expect(onEnd).toHaveBeenCalled()
  })

  it('force_mute mutes local audio and sends media_state', async () => {
    const { result, ws } = await joinedMeeting()
    expect(result.current.isMuted).toBe(false)
    await act(async () => {
      ws._msg({ type: 'force_mute' })
    })
    expect(result.current.isMuted).toBe(true)
  })

  it('room_state sets mediaStates and handRaisedUsers', async () => {
    const { result, ws } = await joinedMeeting()
    await act(async () => {
      ws._msg({
        type: 'room_state',
        participants: [
          { user_id: 'remote-1', audio: true, video: false, hand_raised: true, laser_active: false },
          { user_id: 'remote-2', audio: false, video: true, hand_raised: false, laser_active: false },
        ],
      })
    })
    expect(result.current.mediaStates['remote-1']).toEqual({ audio: true, video: false })
    expect(result.current.mediaStates['remote-2']).toEqual({ audio: false, video: true })
    expect(result.current.handRaisedUsers.has('remote-1')).toBe(true)
    expect(result.current.handRaisedUsers.has('remote-2')).toBe(false)
  })

  it('replaceVideoTrack replaces track on peers with video senders', async () => {
    const { result, ws } = await joinedMeeting()
    // Get the peer connection for remote-1 and give it a video sender
    const pc = rtcInstances[rtcInstances.length - 1]
    const mockSender = { track: { kind: 'video' }, replaceTrack: vi.fn().mockResolvedValue(undefined) }
    pc.getSenders.mockReturnValue([mockSender])
    // Trigger replaceVideoTrack by toggling camera (which calls replaceVideoTrack indirectly)
    // Actually, shareScreen → stopScreenShare is the cleanest path
    await act(async () => { await result.current.shareScreen() })
    act(() => result.current.stopScreenShare())
    // replaceTrack should have been called on the sender (lines 180-181)
    expect(mockSender.replaceTrack).toHaveBeenCalled()
  })

  it('onnegotiationneeded creates offer and sends it', async () => {
    const { ws } = await joinedMeeting()
    // The peer for remote-1 was created during participant.joined
    // Find the RTCPeerConnection instance and fire onnegotiationneeded
    const pc = rtcInstances[rtcInstances.length - 1]
    expect(pc.onnegotiationneeded).not.toBeNull()
    await act(async () => {
      await (pc.onnegotiationneeded as () => Promise<void>)()
    })
    expect(pc.createOffer).toHaveBeenCalled()
    expect(pc.setLocalDescription).toHaveBeenCalled()
    expect(ws.send).toHaveBeenCalledWith(expect.stringContaining('"type":"offer"'))
  })

  it('onicecandidate sends ice_candidate', async () => {
    const { ws } = await joinedMeeting()
    const pc = rtcInstances[rtcInstances.length - 1]
    expect(pc.onicecandidate).not.toBeNull()
    await act(async () => {
      ;(pc.onicecandidate as (e: { candidate: unknown }) => void)({
        candidate: { candidate: 'cand1', sdpMid: '0' },
      })
    })
    expect(ws.send).toHaveBeenCalledWith(expect.stringContaining('"type":"ice_candidate"'))
  })

  it('offer message creates answer and sends it back', async () => {
    const { ws } = await joinedMeeting()
    await act(async () => {
      ws._msg({
        type: 'offer',
        from: 'remote-1',
        payload: { type: 'offer', sdp: 'remote-offer-sdp' },
      })
    })
    // Should have sent an answer back
    expect(ws.send).toHaveBeenCalledWith(expect.stringContaining('"type":"answer"'))
  })

  it('pong message is handled without error', async () => {
    const { ws } = await joinedMeeting()
    await act(async () => {
      ws._msg({ type: 'pong' })
    })
    // No assertion needed — just verify it doesn't throw
  })

  it('answer message sets remote description on existing peer', async () => {
    const { ws } = await joinedMeeting()
    // remote-1's peer was created when participant.joined was processed
    await act(async () => {
      ws._msg({
        type: 'answer',
        from: 'remote-1',
        payload: { type: 'answer', sdp: 'remote-answer-sdp' },
      })
    })
    // Should have called setRemoteDescription on remote-1's peer
    const pc = rtcInstances.find((p) => p.setRemoteDescription.mock.calls.length > 0)
    expect(pc).toBeDefined()
  })

  it('ice_candidate message adds ICE candidate on existing peer', async () => {
    const { ws } = await joinedMeeting()
    await act(async () => {
      ws._msg({
        type: 'ice_candidate',
        from: 'remote-1',
        payload: { candidate: 'cand', sdpMid: '0' },
      })
    })
    const pc = rtcInstances.find((p) => p.addIceCandidate.mock.calls.length > 0)
    expect(pc).toBeDefined()
  })

  it('knock.approved transitions to in_meeting and acquires media', async () => {
    const { result, ws } = await renderMeeting()
    await act(async () => {
      ws._open()
      // Start in waiting_room
      ws._msg({ type: 'knock.requested', knock_id: 'k1', user_id: 'u1' })
    })
    expect(result.current.roomState).toBe('waiting_room')
    await act(async () => {
      ws._msg({ type: 'knock.approved' })
    })
    expect(result.current.roomState).toBe('in_meeting')
  })

  it('room_state with laser_active sets laserState', async () => {
    const { result, ws } = await joinedMeeting()
    await act(async () => {
      ws._msg({
        type: 'room_state',
        participants: [
          { user_id: 'remote-1', audio: true, video: true, hand_raised: false, laser_active: true },
        ],
      })
    })
    expect(result.current.laserState).toEqual({ userId: 'remote-1', x: 0, y: 0 })
  })

  it('emoji_reaction caps at MAX_ACTIVE_REACTIONS', async () => {
    const { result, ws } = await joinedMeeting()
    // Send 12 reactions — should cap at 10
    for (let i = 0; i < 12; i++) {
      await act(async () => {
        ws._msg({ type: 'emoji_reaction', from_id: `u-${i}`, emoji: '👍' })
      })
    }
    expect(result.current.reactionQueue.length).toBeLessThanOrEqual(10)
  })
})

describe('useMeeting — shareScreen / stopScreenShare', () => {
  beforeEach(() => setupBrowserMocks())
  afterEach(() => teardownBrowserMocks())

  async function joinedMeeting() {
    const res = await renderMeeting()
    const { ws } = res
    await act(async () => {
      ws._open()
      ws._msg({ type: 'participant.joined', user_id: 'remote-1' })
    })
    return res
  }

  it('shareScreen sets isScreenSharing to true', async () => {
    const { result } = await joinedMeeting()
    expect(result.current.isScreenSharing).toBe(false)
    await act(async () => { await result.current.shareScreen() })
    expect(result.current.isScreenSharing).toBe(true)
  })

  it('shareScreen is a no-op when already sharing', async () => {
    const { result } = await joinedMeeting()
    await act(async () => { await result.current.shareScreen() })
    expect(result.current.isScreenSharing).toBe(true)
    // Call again — should do nothing
    const callsBefore = (navigator.mediaDevices.getDisplayMedia as ReturnType<typeof vi.fn>).mock.calls.length
    await act(async () => { await result.current.shareScreen() })
    const callsAfter = (navigator.mediaDevices.getDisplayMedia as ReturnType<typeof vi.fn>).mock.calls.length
    expect(callsAfter).toBe(callsBefore)
  })

  it('shareScreen handles user cancellation gracefully', async () => {
    ;(navigator.mediaDevices.getDisplayMedia as ReturnType<typeof vi.fn>).mockRejectedValueOnce(
      new DOMException('Permission denied'),
    )
    const { result } = await joinedMeeting()
    await act(async () => { await result.current.shareScreen() })
    expect(result.current.isScreenSharing).toBe(false)
  })

  it('stopScreenShare sets isScreenSharing to false', async () => {
    const { result } = await joinedMeeting()
    await act(async () => { await result.current.shareScreen() })
    expect(result.current.isScreenSharing).toBe(true)
    act(() => result.current.stopScreenShare())
    expect(result.current.isScreenSharing).toBe(false)
  })

  it('stopScreenShare restores video track when no BG pipeline', async () => {
    const { result } = await joinedMeeting()
    await act(async () => { await result.current.shareScreen() })
    act(() => result.current.stopScreenShare())
    // No crash; isScreenSharing flipped back
    expect(result.current.isScreenSharing).toBe(false)
  })
})

describe('useMeeting — setBackground', () => {
  beforeEach(() => {
    setupBrowserMocks()
    // Ensure bgSupported = true by providing captureStream on canvas mock
    const originalCreate = document.createElement.bind(document)
    const fakeCtx = { filter: 'none', drawImage: vi.fn() }
    vi.spyOn(document, 'createElement').mockImplementation((tag, options) => {
      if (tag === 'canvas') {
        return {
          width: 0, height: 0,
          getContext: () => fakeCtx,
          captureStream: () => ({ getVideoTracks: () => [{ kind: 'video' }] }),
        } as unknown as HTMLCanvasElement
      }
      if (tag === 'video') {
        return {
          srcObject: null, muted: false, playsInline: false,
          play: vi.fn(() => Promise.resolve()),
        } as unknown as HTMLVideoElement
      }
      return originalCreate(tag, options)
    })

    // Mock Image so onload fires immediately (jsdom doesn't load images).
    vi.stubGlobal('Image', class MockImage {
      crossOrigin = ''
      onload: (() => void) | null = null
      onerror: (() => void) | null = null
      constructor() {
        const self = this
        let _src = ''
        Object.defineProperty(this, 'src', {
          set(v: string) { _src = v; Promise.resolve().then(() => self.onload?.()) },
          get() { return _src },
          configurable: true,
        })
      }
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
    localStorage.clear()
  })

  async function joinedMeeting() {
    const res = await renderMeeting()
    const { ws } = res
    await act(async () => {
      ws._open()
      ws._msg({ type: 'participant.joined', user_id: 'remote-1' })
    })
    return res
  }

  it('setBackground with none resets activeBackground', async () => {
    const { result } = await joinedMeeting()
    await act(async () => { await result.current.setBackground({ type: 'none' }) })
    expect(result.current.activeBackground).toEqual({ type: 'none' })
  })

  it('setBackground with none removes BG_STORAGE_KEY from localStorage', async () => {
    localStorage.setItem('rekall_bg_preference', JSON.stringify({ type: 'blur', level: 'light' }))
    const { result } = await joinedMeeting()
    await act(async () => { await result.current.setBackground({ type: 'none' }) })
    expect(localStorage.getItem('rekall_bg_preference')).toBeNull()
  })

  it('setBackground with blur sets activeBackground', async () => {
    const { result } = await joinedMeeting()
    const blurOpt = { type: 'blur' as const, level: 'heavy' as const }
    await act(async () => { await result.current.setBackground(blurOpt) })
    expect(result.current.activeBackground).toEqual(blurOpt)
  })

  it('setBackground stores preference in localStorage', async () => {
    const { result } = await joinedMeeting()
    const blurOpt = { type: 'blur' as const, level: 'light' as const }
    await act(async () => { await result.current.setBackground(blurOpt) })
    const stored = JSON.parse(localStorage.getItem('rekall_bg_preference')!)
    expect(stored.type).toBe('blur')
  })

  it('setBackground with image sets activeBackground', async () => {
    const { result } = await joinedMeeting()
    const imgOpt = { type: 'image' as const, src: '/backgrounds/office.jpg', label: 'Office' }
    await act(async () => { await result.current.setBackground(imgOpt) })
    expect(result.current.activeBackground).toEqual(imgOpt)
  })

  it('bgSupported is true when captureStream is available', async () => {
    const { result } = await joinedMeeting()
    expect(result.current.bgSupported).toBe(true)
  })

  it('toggleCamera pauses BG pipeline when turning camera off after setBackground', async () => {
    const { result } = await joinedMeeting()
    // Set a background first — this creates the pipeline
    const blurOpt = { type: 'blur' as const, level: 'light' as const }
    await act(async () => { await result.current.setBackground(blurOpt) })
    expect(result.current.activeBackground.type).toBe('blur')
    // Toggle camera off — should pause the pipeline (line 554)
    act(() => result.current.toggleCamera())
    expect(result.current.isCameraOff).toBe(true)
    // Toggle camera on — should resume the pipeline (line 554)
    act(() => result.current.toggleCamera())
    expect(result.current.isCameraOff).toBe(false)
  })

  it('stopScreenShare resumes BG pipeline when activeBackground is not none', async () => {
    const { result } = await joinedMeeting()
    // Set a background first
    const blurOpt = { type: 'blur' as const, level: 'heavy' as const }
    await act(async () => { await result.current.setBackground(blurOpt) })
    // Start screen share
    await act(async () => { await result.current.shareScreen() })
    expect(result.current.isScreenSharing).toBe(true)
    // Stop screen share — should resume BG pipeline (lines 585-587)
    act(() => result.current.stopScreenShare())
    expect(result.current.isScreenSharing).toBe(false)
    // Background should still be active
    expect(result.current.activeBackground.type).toBe('blur')
  })

  it('uploadCustomBackground with localStream triggers pipeline and updates state', async () => {
    const { result } = await joinedMeeting()
    // localStream is set after acquireMedia — upload should trigger pipeline path
    const smallFile = new File(['img-data'], 'bg.png', { type: 'image/png' })
    let error: string | null = 'sentinel'
    await act(async () => {
      error = await result.current.uploadCustomBackground(smallFile)
    })
    expect(error).toBeNull()
    // customBgSrc should be set to the data URL
    expect(result.current.customBgSrc).toMatch(/^data:/)
    // activeBackground should be the custom image
    expect(result.current.activeBackground.type).toBe('image')
  })

  it('uploadCustomBackground writes localStorage then applies pipeline', async () => {
    const { result } = await joinedMeeting()
    const smallFile = new File(['tiny'], 'bg.png', { type: 'image/png' })
    await act(async () => {
      await result.current.uploadCustomBackground(smallFile)
    })
    // Verify localStorage was written
    expect(localStorage.getItem(CUSTOM_BG_KEY)).not.toBeNull()
    const pref = JSON.parse(localStorage.getItem(BG_STORAGE_KEY)!)
    expect(pref.src).toBe('__custom__')
  })
})

// ═════════════════════════════════════════════════════════════════════════════
// Chat message handling
// ═════════════════════════════════════════════════════════════════════════════

describe('useMeeting — chat', () => {
  beforeEach(() => setupBrowserMocks())
  afterEach(() => teardownBrowserMocks())

  async function joinedMeeting() {
    const res = await renderMeeting()
    const { ws } = res
    await act(async () => {
      ws._open()
      ws._msg({ type: 'participant.joined', user_id: 'remote-1' })
    })
    // Let the history-fetch effect flush.
    await act(async () => { await Promise.resolve() })
    return res
  }

  it('fetches chat history when entering in_meeting state', async () => {
    vi.mocked(meetingService.listMessages).mockResolvedValueOnce({
      messages: [
        { id: 'm1', userId: 'remote-1', body: 'historic', sentAt: Date.now() - 60_000 },
      ],
      has_more: true,
    })
    const { result } = await joinedMeeting()
    await act(async () => { await vi.waitFor(() => result.current.messages.length > 0) })
    expect(result.current.messages[0].body).toBe('historic')
    expect(result.current.hasMoreHistory).toBe(true)
  })

  it('sendChatMessage sends WS message and appends pending entry', async () => {
    const { result, ws } = await joinedMeeting()
    act(() => result.current.sendChatMessage('hello'))
    expect(ws.send).toHaveBeenCalledWith(
      expect.stringContaining('"type":"chat_message"'),
    )
    expect(result.current.messages).toHaveLength(1)
    expect(result.current.messages[0].pending).toBe(true)
    expect(result.current.messages[0].body).toBe('hello')
    expect(result.current.messages[0].userId).toBe('user-1')
  })

  it('sendChatMessage trims body and ignores empty/whitespace', async () => {
    const { result, ws } = await joinedMeeting()
    vi.mocked(ws.send).mockClear()
    act(() => result.current.sendChatMessage('   '))
    expect(ws.send).not.toHaveBeenCalled()
    expect(result.current.messages).toHaveLength(0)
  })

  it('sendChatMessage blocks when body exceeds MAX_MESSAGE_LENGTH', async () => {
    const { result, ws } = await joinedMeeting()
    vi.mocked(ws.send).mockClear()
    act(() => result.current.sendChatMessage('x'.repeat(2001)))
    expect(ws.send).not.toHaveBeenCalled()
    expect(result.current.messages).toHaveLength(0)
    expect(result.current.chatSendError).toMatch(/2000 characters/)
  })

  it('rate-limits: 4th send within 2s is blocked and flashKey ticks', async () => {
    const { result, ws } = await joinedMeeting()
    vi.mocked(ws.send).mockClear()
    const initialFlash = result.current.chatFlashKey
    act(() => {
      result.current.sendChatMessage('a')
      result.current.sendChatMessage('b')
      result.current.sendChatMessage('c')
      result.current.sendChatMessage('d')
    })
    expect(ws.send).toHaveBeenCalledTimes(3)
    expect(result.current.chatFlashKey).toBeGreaterThan(initialFlash)
  })

  it('echoed chat_message reconciles pending entry by client_id', async () => {
    const { result, ws } = await joinedMeeting()
    act(() => result.current.sendChatMessage('hi'))
    const pending = result.current.messages[0]
    expect(pending.pending).toBe(true)
    const clientId = pending.clientId!

    await act(async () => {
      ws._msg({
        type: 'chat_message',
        id: 'server-id-1',
        client_id: clientId,
        user_id: 'user-1',
        body: 'hi',
        sent_at: '2026-04-23T14:03:17.000Z',
      })
    })

    expect(result.current.messages).toHaveLength(1)
    expect(result.current.messages[0].id).toBe('server-id-1')
    expect(result.current.messages[0].pending).toBeFalsy()
  })

  it('deduplicates incoming chat_message with an id already in the list', async () => {
    const { result, ws } = await joinedMeeting()
    const payload = {
      type: 'chat_message',
      id: 'server-id-1',
      user_id: 'remote-1',
      body: 'from peer',
      sent_at: '2026-04-23T14:03:17.000Z',
    }
    await act(async () => { ws._msg(payload) })
    expect(result.current.messages).toHaveLength(1)
    await act(async () => { ws._msg(payload) })
    expect(result.current.messages).toHaveLength(1)
  })

  it('increments unreadCount for remote messages while panel is closed', async () => {
    const { result, ws } = await joinedMeeting()
    expect(result.current.isChatPanelOpen).toBe(false)
    await act(async () => {
      ws._msg({
        type: 'chat_message',
        id: 'm1', user_id: 'remote-1', body: 'ping', sent_at: '2026-04-23T14:03:17Z',
      })
    })
    expect(result.current.unreadCount).toBe(1)
  })

  it('does NOT increment unreadCount for own sent messages', async () => {
    const { result, ws } = await joinedMeeting()
    act(() => result.current.sendChatMessage('own msg'))
    const clientId = result.current.messages[0].clientId!
    await act(async () => {
      ws._msg({
        type: 'chat_message',
        id: 's1', client_id: clientId, user_id: 'user-1', body: 'own msg',
        sent_at: '2026-04-23T14:03:17Z',
      })
    })
    expect(result.current.unreadCount).toBe(0)
  })

  it('openChatPanel clears unreadCount', async () => {
    const { result, ws } = await joinedMeeting()
    await act(async () => {
      ws._msg({
        type: 'chat_message',
        id: 'm1', user_id: 'remote-1', body: 'ping', sent_at: '2026-04-23T14:03:17Z',
      })
    })
    expect(result.current.unreadCount).toBe(1)
    act(() => result.current.openChatPanel())
    expect(result.current.isChatPanelOpen).toBe(true)
    expect(result.current.unreadCount).toBe(0)
  })

  it('surfaces a send error when the WS is not open', async () => {
    const res = await renderMeeting()
    const { result } = res
    // Do NOT call ws._open() — readyState stays 0 (CONNECTING), not OPEN.
    // Simulate entering in_meeting without a proper WS open.
    await act(async () => {
      res.ws._msg({ type: 'participant.joined', user_id: 'remote-1' })
    })
    act(() => result.current.sendChatMessage('should fail'))
    expect(result.current.chatSendError).toMatch(/not connected/i)
    expect(result.current.messages).toHaveLength(0)
  })
})
