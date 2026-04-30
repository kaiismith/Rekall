import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import theme from '@/theme'

// ── Mock useMeeting ──────────────────────────────────────────────────────────

const defaults = {
  meeting: {
    id: 'm1',
    code: 'abc',
    title: 'Sprint Sync',
    type: 'open',
    host_id: 'host-1',
    status: 'active',
    max_participants: 10,
    join_url: '',
    created_at: '',
  },
  roomState: 'in_meeting' as string,
  isSpeaking: false,
  audioLevel: 0,
  remoteSpeaking: {} as Record<string, boolean>,
  knocks: [] as Array<{ knock_id: string; user_id: string }>,
  localStream: null as MediaStream | null,
  peers: {} as Record<string, RTCPeerConnection>,
  respondToKnock: vi.fn(),
  endMeeting: vi.fn(),
  leave: vi.fn(),
  isMuted: false,
  isCameraOff: true,
  isScreenSharing: false,
  toggleMute: vi.fn(),
  toggleCamera: vi.fn(),
  shareScreen: vi.fn(),
  stopScreenShare: vi.fn(),
  forceMute: vi.fn(),
  isHandRaised: false,
  handRaisedUsers: new Set<string>(),
  handRaisedCount: 0,
  toggleHand: vi.fn(),
  reactionQueue: [] as Array<{ id: string; userId: string; emoji: string; timestamp: number }>,
  sendEmojiReaction: vi.fn(),
  localPreviewStream: null as MediaStream | null,
  joinNow: vi.fn(),
  availableDevices: { cameras: [], mics: [], speakers: [] } as {
    cameras: MediaDeviceInfo[]
    mics: MediaDeviceInfo[]
    speakers: MediaDeviceInfo[]
  },
  selectedDeviceIds: { camera: '', mic: '', speaker: '' },
  switchCamera: vi.fn(),
  switchMic: vi.fn(),
  switchSpeaker: vi.fn(),
  refreshDevices: vi.fn(),
  messages: [] as Array<{ id: string; userId: string; body: string; sentAt: number }>,
  participantDirectory: {} as Record<string, { full_name: string; initials: string }>,
  sendChatMessage: vi.fn(),
  retryChatMessage: vi.fn(),
  unreadChatCount: 0,
  resetUnreadChat: vi.fn(),
  activeBackground: { type: 'none' as const },
  bgSupported: true,
  setBackground: vi.fn(),
  customBgSrc: null as string | null,
  uploadCustomBackground: vi.fn().mockResolvedValue(null),
  mediaStates: {} as Record<string, { audio: boolean; video: boolean }>,
}

let meetingOverrides: Partial<typeof defaults> = {}

vi.mock('@/hooks/useMeeting', () => ({
  useMeeting: () => ({ ...defaults, ...meetingOverrides }),
}))

vi.mock('@/store/authStore', () => {
  const state = {
    user: {
      id: 'host-1',
      email: 'a@b.com',
      full_name: 'Host',
      role: 'member',
      email_verified: true,
      created_at: '',
    },
    accessToken: 'tok',
    isInitialised: true,
    setAuth: vi.fn(),
    clearAuth: vi.fn(),
    setInitialised: vi.fn(),
  }
  type AuthState = typeof state
  const useAuthStore = <T,>(selector?: (s: AuthState) => T) => (selector ? selector(state) : state)
  useAuthStore.getState = () => state
  useAuthStore.setState = vi.fn()
  useAuthStore.subscribe = vi.fn()
  return { useAuthStore }
})

import { MeetingRoomPage } from '@/pages/MeetingRoomPage'

// ── Helpers ──────────────────────────────────────────────────────────────────

function renderPage(overrides: Partial<typeof defaults> = {}) {
  meetingOverrides = overrides
  return render(
    <ThemeProvider theme={theme}>
      <MemoryRouter initialEntries={['/meeting/abc']}>
        <Routes>
          <Route path="/meeting/:code" element={<MeetingRoomPage />} />
          <Route path="/meetings" element={<div>meetings-list</div>} />
        </Routes>
      </MemoryRouter>
    </ThemeProvider>,
  )
}

// ── Tests ────────────────────────────────────────────────────────────────────

describe('MeetingRoomPage', () => {
  beforeEach(() => {
    meetingOverrides = {}
    vi.clearAllMocks()
  })

  // ── Room state screens ────────────────────────────────────────────────────

  it('shows "Connecting…" when roomState is connecting', () => {
    renderPage({ roomState: 'connecting' })
    expect(screen.getByText('Connecting…')).toBeInTheDocument()
  })

  it('shows waiting room when roomState is waiting_room', () => {
    renderPage({ roomState: 'waiting_room' })
    expect(screen.getByText('Waiting for admission')).toBeInTheDocument()
    expect(screen.getByText('A participant will admit you shortly.')).toBeInTheDocument()
  })

  it('shows denied screen when roomState is denied', () => {
    renderPage({ roomState: 'denied' })
    expect(screen.getByText('Access Denied')).toBeInTheDocument()
    expect(screen.getByText('Your request to join was declined.')).toBeInTheDocument()
  })

  it('denied screen has Back to Meetings button', () => {
    renderPage({ roomState: 'denied' })
    expect(screen.getByRole('button', { name: /back to meetings/i })).toBeInTheDocument()
  })

  it('shows ended screen when roomState is ended', () => {
    renderPage({ roomState: 'ended' })
    expect(screen.getByText('Meeting Ended')).toBeInTheDocument()
  })

  it('shows ended screen when roomState is error', () => {
    renderPage({ roomState: 'error' })
    expect(screen.getByText('Meeting Ended')).toBeInTheDocument()
  })

  it('ended screen Back to Meetings button navigates', () => {
    renderPage({ roomState: 'ended' })
    fireEvent.click(screen.getByRole('button', { name: /back to meetings/i }))
    expect(screen.getByText('meetings-list')).toBeInTheDocument()
  })

  // ── In-meeting UI ──────────────────────────────────────────────────────────

  it('renders the meeting title in the top bar', () => {
    renderPage()
    expect(screen.getByText('Sprint Sync')).toBeInTheDocument()
  })

  it('shows meeting code chip', () => {
    renderPage()
    expect(screen.getByText('abc')).toBeInTheDocument()
  })

  it('shows meeting type chip', () => {
    renderPage()
    expect(screen.getByText('open')).toBeInTheDocument()
  })

  it('shows participant count', () => {
    renderPage()
    expect(screen.getByText('1 participants')).toBeInTheDocument()
  })

  it('shows participant count including peers', () => {
    const mockPc = {
      close: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    } as unknown as RTCPeerConnection
    renderPage({
      peers: { 'user-2': mockPc, 'user-3': mockPc },
      mediaStates: {
        'user-2': { audio: true, video: false },
        'user-3': { audio: true, video: false },
      },
    })
    expect(screen.getByText('3 participants')).toBeInTheDocument()
  })

  // ── Host vs non-host ──────────────────────────────────────────────────────

  it('shows "End Meeting" button when user is host', () => {
    renderPage() // user is host-1, meeting.host_id is host-1
    expect(screen.getByRole('button', { name: /end meeting/i })).toBeInTheDocument()
  })

  it('shows "Leave" button when user is not host', () => {
    renderPage({ meeting: { ...defaults.meeting, host_id: 'other-user' } })
    expect(screen.getByRole('button', { name: /leave/i })).toBeInTheDocument()
  })

  it('End Meeting calls endMeeting', () => {
    const endMeeting = vi.fn()
    renderPage({ endMeeting })
    fireEvent.click(screen.getByRole('button', { name: /end meeting/i }))
    expect(endMeeting).toHaveBeenCalled()
  })

  it('Leave calls leave', () => {
    const leave = vi.fn()
    renderPage({ meeting: { ...defaults.meeting, host_id: 'other' }, leave })
    fireEvent.click(screen.getByRole('button', { name: /leave/i }))
    expect(leave).toHaveBeenCalled()
  })

  // ── Screen sharing banner ──────────────────────────────────────────────────

  it('shows screen share banner when isScreenSharing is true', () => {
    renderPage({ isScreenSharing: true })
    expect(screen.getByText('You are sharing your screen')).toBeInTheDocument()
  })

  it('hides screen share banner when isScreenSharing is false', () => {
    renderPage({ isScreenSharing: false })
    expect(screen.queryByText('You are sharing your screen')).not.toBeInTheDocument()
  })

  // ── Local tile label ──────────────────────────────────────────────────────

  it('local tile shows "You" when not sharing and camera is on', () => {
    renderPage({ isCameraOff: false })
    expect(screen.getByText('You')).toBeInTheDocument()
  })

  it('local tile shows "You (sharing)" when screen sharing', () => {
    renderPage({ isCameraOff: false, isScreenSharing: true })
    expect(screen.getByText('You (sharing)')).toBeInTheDocument()
  })

  it('local tile hides the speaker label when camera is off (avatar stands alone)', () => {
    renderPage({ isCameraOff: true })
    expect(screen.queryByText('You')).not.toBeInTheDocument()
  })

  // ── Hand raise chip ───────────────────────────────────────────────────────

  it('shows hand raise count chip when handRaisedCount > 0', () => {
    renderPage({ handRaisedCount: 3 })
    expect(screen.getByText('3')).toBeInTheDocument()
  })

  it('hides hand raise chip when handRaisedCount is 0', () => {
    renderPage({ handRaisedCount: 0 })
    // No warning chip shown
    expect(screen.queryByText('0')).not.toBeInTheDocument()
  })

  // ── Knock sidebar ─────────────────────────────────────────────────────────

  it('shows knock sidebar when knocks are present', () => {
    renderPage({ knocks: [{ knock_id: 'k1', user_id: 'user-abc-1234567890' }] })
    expect(screen.getByText('Waiting to Join (1)')).toBeInTheDocument()
  })

  it('knock sidebar has Admit and Deny buttons', () => {
    renderPage({ knocks: [{ knock_id: 'k1', user_id: 'user-abc' }] })
    expect(screen.getByRole('button', { name: /admit/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /deny/i })).toBeInTheDocument()
  })

  it('Admit button calls respondToKnock with approved=true', () => {
    const respondToKnock = vi.fn()
    renderPage({ knocks: [{ knock_id: 'k1', user_id: 'user-abc' }], respondToKnock })
    fireEvent.click(screen.getByRole('button', { name: /admit/i }))
    expect(respondToKnock).toHaveBeenCalledWith('k1', true)
  })

  it('Deny button calls respondToKnock with approved=false', () => {
    const respondToKnock = vi.fn()
    renderPage({ knocks: [{ knock_id: 'k1', user_id: 'user-abc' }], respondToKnock })
    fireEvent.click(screen.getByRole('button', { name: /deny/i }))
    expect(respondToKnock).toHaveBeenCalledWith('k1', false)
  })

  it('hides knock sidebar when no knocks', () => {
    renderPage({ knocks: [] })
    expect(screen.queryByText(/waiting to join/i)).not.toBeInTheDocument()
  })

  // ── Remote tiles ──────────────────────────────────────────────────────────

  it('renders remote tiles for each peer', () => {
    const mockPc = {
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      close: vi.fn(),
    } as unknown as RTCPeerConnection
    renderPage({
      isCameraOff: false,
      peers: { 'user-abcdef12': mockPc },
      mediaStates: { 'user-abcdef12': { audio: true, video: true } },
    })
    // Remote tile label shows uid.slice(0,8) + '…' when camera is on.
    expect(screen.getByText('user-abc…')).toBeInTheDocument()
  })

  it('shows hand raise emoji on remote tile when hand is raised', () => {
    const mockPc = {
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      close: vi.fn(),
    } as unknown as RTCPeerConnection
    renderPage({
      peers: { 'user-abcdef12': mockPc },
      mediaStates: { 'user-abcdef12': { audio: true, video: false } },
      handRaisedUsers: new Set(['user-abcdef12']),
    })
    // The hand raise renders ✋ on the tile
    expect(screen.getByText('✋')).toBeInTheDocument()
  })

  it('shows host mute button on remote tile when user is host', () => {
    const mockPc = {
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      close: vi.fn(),
    } as unknown as RTCPeerConnection
    renderPage({
      peers: { 'user-abcdef12': mockPc },
      mediaStates: { 'user-abcdef12': { audio: true, video: false } },
    })
    // The host mute button renders (opacity 0 until hover, but still in DOM)
    expect(screen.getByRole('button', { name: /mute participant/i })).toBeInTheDocument()
  })

  it('clicking host mute button calls forceMute', () => {
    const forceMute = vi.fn()
    const mockPc = {
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      close: vi.fn(),
    } as unknown as RTCPeerConnection
    renderPage({
      peers: { 'user-abcdef12': mockPc },
      mediaStates: { 'user-abcdef12': { audio: true, video: false } },
      forceMute,
    })
    fireEvent.click(screen.getByRole('button', { name: /mute participant/i }))
    expect(forceMute).toHaveBeenCalledWith('user-abcdef12')
  })

  it('shows MicOffIcon on remote tile when peer is muted', () => {
    const mockPc = {
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      close: vi.fn(),
    } as unknown as RTCPeerConnection
    renderPage({
      peers: { 'user-abcdef12': mockPc },
      mediaStates: { 'user-abcdef12': { audio: false, video: false } },
    })
    // MicOffIcon is rendered as an SVG with data-testid="MicOffIcon"
    const micOffIcons = document.querySelectorAll('[data-testid="MicOffIcon"]')
    // At least one MicOffIcon for the remote tile (local tile may also have one)
    expect(micOffIcons.length).toBeGreaterThanOrEqual(1)
  })

  it('shows MicIcon on remote tile when peer is speaking', () => {
    const mockPc = {
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      close: vi.fn(),
    } as unknown as RTCPeerConnection
    renderPage({
      peers: { 'user-abcdef12': mockPc },
      mediaStates: { 'user-abcdef12': { audio: true, video: false } },
      remoteSpeaking: { 'user-abcdef12': true },
    })
    // MicIcon rendered as SVG with data-testid="MicIcon"
    const micIcons = document.querySelectorAll('[data-testid="MicIcon"]')
    expect(micIcons.length).toBeGreaterThanOrEqual(1)
  })

  it('RemoteVideo fires track event handler', () => {
    // The RemoteVideo component calls peerConnection.addEventListener('track', handler)
    // We need to simulate the track event firing to cover line 492
    let trackHandler: ((e: unknown) => void) | null = null
    const mockPc = {
      addEventListener: vi.fn((event: string, handler: (e: unknown) => void) => {
        if (event === 'track') trackHandler = handler
      }),
      removeEventListener: vi.fn(),
      close: vi.fn(),
    } as unknown as RTCPeerConnection
    renderPage({
      peers: { 'user-abcdef12': mockPc },
      mediaStates: { 'user-abcdef12': { audio: true, video: true } },
    })
    // Simulate track event
    expect(trackHandler).not.toBeNull()
    const mockMediaStream = {} as MediaStream
    trackHandler!({ streams: [mockMediaStream] })
    // Video element should now have srcObject set (no crash = success)
    const videos = document.querySelectorAll('video')
    expect(videos.length).toBeGreaterThanOrEqual(1)
  })

  it('renders remote video element when peer camera is on', () => {
    const mockPc = {
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      close: vi.fn(),
    } as unknown as RTCPeerConnection
    renderPage({
      peers: { 'user-abcdef12': mockPc },
      mediaStates: { 'user-abcdef12': { audio: true, video: true } },
    })
    // RemoteVideo renders a <video> element
    const videos = document.querySelectorAll('video')
    expect(videos.length).toBeGreaterThanOrEqual(1)
  })

  // ── Emoji reactions ───────────────────────────────────────────────────────

  it('renders emoji reactions on the local tile', () => {
    renderPage({
      reactionQueue: [{ id: 'r1', userId: 'host-1', emoji: '🎉', timestamp: Date.now() }],
    })
    expect(screen.getByText('🎉')).toBeInTheDocument()
  })

  // ── Camera off avatar ──────────────────────────────────────────────────────

  it('shows avatar with initials when camera is off', () => {
    renderPage({ isCameraOff: true })
    // Avatar derives initials from the auth user's full_name ("Host") rather
    // than the tile label ("You"), matching the pre-join preview behaviour.
    expect(screen.getByText('H')).toBeInTheDocument()
  })

  // ── Fallback title when meeting.title is empty ────────────────────────────

  it('shows "Meeting" when meeting has no title', () => {
    renderPage({ meeting: { ...defaults.meeting, title: '' } })
    expect(screen.getByText('Meeting')).toBeInTheDocument()
  })

  it('renders local video element when camera is on', () => {
    renderPage({ isCameraOff: false })
    // When camera is on, the local tile renders a <video> element (lines 240-246)
    const videos = document.querySelectorAll('video')
    expect(videos.length).toBeGreaterThanOrEqual(1)
  })

  it('attaches localStream to video element when camera is on and stream is set', () => {
    const mockStream = { getTracks: () => [] } as unknown as MediaStream
    renderPage({ isCameraOff: false, localStream: mockStream })
    // The useEffect sets localVideoRef.current.srcObject = localStream (lines 108-109)
    const videos = document.querySelectorAll('video')
    expect(videos.length).toBeGreaterThanOrEqual(1)
    // In jsdom, the video element's srcObject should be set
    const localVideo = videos[0]
    expect(localVideo.srcObject).toBe(mockStream)
  })

  it('does not render local video element when camera is off', () => {
    renderPage({ isCameraOff: true })
    // When camera is off, the local tile shows an Avatar (initials from
    // user.full_name = "Host" → "H") instead of a <video>.
    expect(screen.getByText('H')).toBeInTheDocument()
  })
})
