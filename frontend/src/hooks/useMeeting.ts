import { useCallback, useEffect, useRef, useState } from 'react'
import { useAuthStore } from '@/store/authStore'
import { meetingService } from '@/services/meetingService'
import { VadDetector } from '@/utils/vadDetector'
import { VirtualBackgroundPipeline } from '@/utils/virtualBackgroundPipeline'
import type {
  Meeting,
  MeetingRoomState,
  WsMessage,
  KnockEntry,
  MediaState,
  EmojiReaction,
  LaserState,
  BackgroundOption,
  ChatMessage,
  ParticipantDirectoryEntry,
} from '@/types/meeting'
import {
  MAX_MESSAGE_LENGTH,
  RATE_LIMIT_MESSAGES,
  RATE_LIMIT_WINDOW_MS,
  HISTORY_PAGE_SIZE,
  PENDING_MESSAGE_TIMEOUT_MS,
} from '@/config/chat'

const LASER_THROTTLE_MS = 33  // ~30 fps
const EMOJI_RATE_LIMIT_MS = 500
const MAX_ACTIVE_REACTIONS = 10
const BG_STORAGE_KEY = 'rekall_bg_preference'
const CUSTOM_BG_KEY = 'rekall_bg_custom'
const CUSTOM_SRC_SENTINEL = '__custom__'
const CUSTOM_BG_MAX_BYTES = 2 * 1024 * 1024 // 2 MB file size limit

export interface UseMeetingOptions {
  code: string
  onEnd?: () => void
}

export interface UseMeetingReturn {
  // ── core ────────────────────────────────────────────────────────────────────
  meeting: Meeting | null
  roomState: MeetingRoomState
  isSpeaking: boolean
  audioLevel: number
  remoteSpeaking: Record<string, boolean>
  knocks: KnockEntry[]
  localStream: MediaStream | null
  peers: Record<string, RTCPeerConnection>
  respondToKnock: (knockId: string, approved: boolean) => void
  endMeeting: () => Promise<void>
  leave: () => void
  // ── media controls ──────────────────────────────────────────────────────────
  isMuted: boolean
  isCameraOff: boolean
  isScreenSharing: boolean
  toggleMute: () => void
  toggleCamera: () => void
  shareScreen: () => Promise<void>
  stopScreenShare: () => void
  forceMute: (userId: string) => void
  // ── hand raise ──────────────────────────────────────────────────────────────
  isHandRaised: boolean
  handRaisedUsers: Set<string>
  handRaisedCount: number
  toggleHand: () => void
  // ── emoji reactions ─────────────────────────────────────────────────────────
  reactionQueue: EmojiReaction[]
  sendEmojiReaction: (emoji: string) => void
  // ── laser pointer ───────────────────────────────────────────────────────────
  laserState: LaserState | null
  isLaserActive: boolean
  toggleLaser: () => void
  sendLaserMove: (x: number, y: number) => void
  // ── virtual background ──────────────────────────────────────────────────────
  activeBackground: BackgroundOption
  bgSupported: boolean
  setBackground: (option: BackgroundOption) => Promise<void>
  customBgSrc: string | null
  /** Upload a custom background image. Returns an error string on failure, null on success. */
  uploadCustomBackground: (file: File) => Promise<string | null>
  // ── remote participant state ─────────────────────────────────────────────────
  mediaStates: Record<string, MediaState>
  // ── chat ────────────────────────────────────────────────────────────────────
  messages: ChatMessage[]
  unreadCount: number
  isChatPanelOpen: boolean
  isLoadingHistory: boolean
  hasMoreHistory: boolean
  chatHistoryError: string | null
  participantDirectory: Record<string, ParticipantDirectoryEntry>
  chatFlashKey: number
  chatSendError: string | null
  openChatPanel: () => void
  closeChatPanel: () => void
  sendChatMessage: (body: string) => void
  retrySendMessage: (localId: string) => void
  deleteFailedMessage: (localId: string) => void
  loadOlderMessages: () => Promise<void>
  retryHistoryFetch: () => void
  dismissChatSendError: () => void
}

const RTC_CONFIG: RTCConfiguration = {
  iceServers: [{ urls: 'stun:stun.l.google.com:19302' }],
}

function generateId(): string {
  return crypto.randomUUID()
}

function loadStoredBackground(): BackgroundOption {
  try {
    const raw = localStorage.getItem(BG_STORAGE_KEY)
    if (!raw) return { type: 'none' }
    const opt = JSON.parse(raw) as BackgroundOption
    // Resolve custom image sentinel → actual data URL.
    if (opt.type === 'image' && opt.src === CUSTOM_SRC_SENTINEL) {
      const src = localStorage.getItem(CUSTOM_BG_KEY)
      if (!src) return { type: 'none' }
      return { type: 'image', src, label: 'Custom' }
    }
    return opt
  } catch { return { type: 'none' } }
}

function loadCustomBgSrc(): string | null {
  try { return localStorage.getItem(CUSTOM_BG_KEY) } catch { return null }
}

export function useMeeting({ code, onEnd }: UseMeetingOptions): UseMeetingReturn {
  const { accessToken, user } = useAuthStore()

  // ── core state ─────────────────────────────────────────────────────────────
  const [meeting, setMeeting] = useState<Meeting | null>(null)
  const [roomState, setRoomState] = useState<MeetingRoomState>('connecting')
  const [isSpeaking, setIsSpeaking] = useState(false)
  const [audioLevel, setAudioLevel] = useState(0)
  const [remoteSpeaking, setRemoteSpeaking] = useState<Record<string, boolean>>({})
  const [knocks, setKnocks] = useState<KnockEntry[]>([])
  const [localStream, setLocalStream] = useState<MediaStream | null>(null)
  const [peers, setPeers] = useState<Record<string, RTCPeerConnection>>({})

  // ── media control state ────────────────────────────────────────────────────
  const [isMuted, setIsMuted] = useState(false)
  const [isCameraOff, setIsCameraOff] = useState(false)
  const [isScreenSharing, setIsScreenSharing] = useState(false)

  // ── hand + emoji + laser state ─────────────────────────────────────────────
  const [isHandRaised, setIsHandRaised] = useState(false)
  const [handRaisedUsers, setHandRaisedUsers] = useState<Set<string>>(new Set())
  const [reactionQueue, setReactionQueue] = useState<EmojiReaction[]>([])
  const [laserState, setLaserState] = useState<LaserState | null>(null)
  const [isLaserActive, setIsLaserActive] = useState(false)

  // ── virtual background state ───────────────────────────────────────────────
  const [activeBackground, setActiveBackgroundState] = useState<BackgroundOption>(loadStoredBackground)
  const [bgSupported] = useState(() => VirtualBackgroundPipeline.isSupported())
  const [customBgSrc, setCustomBgSrc] = useState<string | null>(loadCustomBgSrc)

  // ── remote participant media state (from room_state / media_state events) ──
  const [mediaStates, setMediaStates] = useState<Record<string, MediaState>>({})

  // ── chat state ─────────────────────────────────────────────────────────────
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [unreadCount, setUnreadCount] = useState(0)
  const [isChatPanelOpen, setIsChatPanelOpen] = useState(false)
  const [isLoadingHistory, setIsLoadingHistory] = useState(false)
  const [hasMoreHistory, setHasMoreHistory] = useState(false)
  const [chatHistoryError, setChatHistoryError] = useState<string | null>(null)
  const [participantDirectory, setParticipantDirectory] = useState<Record<string, ParticipantDirectoryEntry>>({})
  const [chatFlashKey, setChatFlashKey] = useState(0)
  const [chatSendError, setChatSendError] = useState<string | null>(null)

  // ── refs ───────────────────────────────────────────────────────────────────
  const wsRef = useRef<WebSocket | null>(null)
  const vadRef = useRef<VadDetector | null>(null)
  const peersRef = useRef<Record<string, RTCPeerConnection>>({})
  const bgPipelineRef = useRef<VirtualBackgroundPipeline | null>(null)
  const localTracksRef = useRef<{
    audioTrack: MediaStreamTrack | null
    videoTrack: MediaStreamTrack | null   // original camera track
    screenTrack: MediaStreamTrack | null  // screen capture track
    canvasTrack: MediaStreamTrack | null  // virtual BG canvas track
  }>({ audioTrack: null, videoTrack: null, screenTrack: null, canvasTrack: null })
  const lastLaserSentRef = useRef(0)
  const lastEmojiSentRef = useRef(0)
  const chatSendTimestampsRef = useRef<number[]>([])
  const localUserIdRef = useRef<string | null>(null)
  const isChatPanelOpenRef = useRef(false)
  // Tracks setTimeout handles keyed by client_id so we can clear them on echo
  // and mark the message `failed` if the echo never arrives.
  const pendingMsgTimeoutsRef = useRef<Record<string, ReturnType<typeof setTimeout>>>({})
  // Keep a ref to isMuted/isCameraOff for use inside callbacks without stale closures.
  const isMutedRef = useRef(false)
  const isCameraOffRef = useRef(false)
  const isScreenSharingRef = useRef(false)
  const isHandRaisedRef = useRef(false)
  const isLaserActiveRef = useRef(false)

  // Sync refs with state.
  useEffect(() => { isMutedRef.current = isMuted }, [isMuted])
  useEffect(() => { isCameraOffRef.current = isCameraOff }, [isCameraOff])
  useEffect(() => { isScreenSharingRef.current = isScreenSharing }, [isScreenSharing])
  useEffect(() => { isHandRaisedRef.current = isHandRaised }, [isHandRaised])
  useEffect(() => { isLaserActiveRef.current = isLaserActive }, [isLaserActive])
  useEffect(() => { localUserIdRef.current = user?.id ?? null }, [user])
  useEffect(() => { isChatPanelOpenRef.current = isChatPanelOpen }, [isChatPanelOpen])

  // ── WS send helper ─────────────────────────────────────────────────────────
  const send = useCallback((msg: WsMessage) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(msg))
    }
  }, [])

  // ── Active video track priority: screen > canvas > camera ─────────────────
  const getActiveVideoTrack = useCallback((): MediaStreamTrack | null => {
    const t = localTracksRef.current
    return t.screenTrack ?? t.canvasTrack ?? t.videoTrack
  }, [])

  const replaceVideoTrack = useCallback((newTrack: MediaStreamTrack | null) => {
    Object.values(peersRef.current).forEach((pc) => {
      const sender = pc.getSenders().find((s) => s.track?.kind === 'video')
      if (sender && newTrack) {
        sender.replaceTrack(newTrack).catch(() => { /* connection may be closed */ })
      }
    })
  }, [])

  // ── Peer connection helpers ────────────────────────────────────────────────
  const getPeer = useCallback((userId: string): RTCPeerConnection => {
    if (peersRef.current[userId]) return peersRef.current[userId]

    const pc = new RTCPeerConnection(RTC_CONFIG)

    pc.onicecandidate = (e) => {
      if (e.candidate) {
        send({ type: 'ice_candidate', to: userId, payload: e.candidate })
      }
    }

    pc.onnegotiationneeded = async () => {
      const offer = await pc.createOffer()
      await pc.setLocalDescription(offer)
      send({ type: 'offer', to: userId, payload: offer })
    }

    peersRef.current[userId] = pc
    setPeers((prev) => ({ ...prev, [userId]: pc }))
    return pc
  }, [send])

  const removePeer = useCallback((userId: string) => {
    peersRef.current[userId]?.close()
    delete peersRef.current[userId]
    setPeers((prev) => {
      const next = { ...prev }
      delete next[userId]
      return next
    })
  }, [])

  // ── VAD ───────────────────────────────────────────────────────────────────
  const startVad = useCallback((stream: MediaStream) => {
    vadRef.current = new VadDetector(({ speaking, level }) => {
      setIsSpeaking(speaking)
      setAudioLevel(level)
      send({ type: 'speaking_state', payload: { speaking } })
    })
    vadRef.current.start(stream)
  }, [send])

  // ── Media acquisition ─────────────────────────────────────────────────────
  const acquireMedia = useCallback(async (): Promise<MediaStream> => {
    const stream = await navigator.mediaDevices.getUserMedia({
      audio: { echoCancellation: true, noiseSuppression: true, autoGainControl: true },
      video: { width: 1280, height: 720, frameRate: 30 },
    })
    const [audioTrack] = stream.getAudioTracks()
    const [videoTrack] = stream.getVideoTracks()
    localTracksRef.current.audioTrack = audioTrack ?? null
    localTracksRef.current.videoTrack = videoTrack ?? null

    setLocalStream(stream)
    startVad(stream)

    // Re-apply stored background if user had one set.
    const stored = loadStoredBackground()
    if (stored.type !== 'none' && bgSupported) {
      // Non-blocking — errors are swallowed; raw camera used as fallback.
      void (async () => {
        try {
          const pipeline = new VirtualBackgroundPipeline(stream)
          bgPipelineRef.current = pipeline
          const canvasTrack = await pipeline.setBackground(stored)
          if (canvasTrack) {
            localTracksRef.current.canvasTrack = canvasTrack
            setActiveBackgroundState(stored)
          }
        } catch { /* ignore */ }
      })()
    }

    return stream
  }, [startVad, bgSupported])

  // ── Track helper: add local tracks to a peer connection ───────────────────
  const addLocalTracksToPeer = useCallback((pc: RTCPeerConnection, stream: MediaStream) => {
    const activeVideo = getActiveVideoTrack()
    stream.getAudioTracks().forEach((t) => pc.addTrack(t, stream))
    if (activeVideo) {
      pc.addTrack(activeVideo, stream)
    } else {
      stream.getVideoTracks().forEach((t) => pc.addTrack(t, stream))
    }
  }, [getActiveVideoTrack])

  // ── WS message handler ────────────────────────────────────────────────────
  const handleMessage = useCallback(async (msg: WsMessage) => {
    switch (msg.type) {
      case 'participant.joined': {
        if (!msg.user_id) break
        // Record display info so chat messages from this user render with a
        // name immediately — no per-user fetch required.
        if (msg.full_name || msg.initials) {
          setParticipantDirectory(prev => (
            prev[msg.user_id!]
              ? prev
              : { ...prev, [msg.user_id!]: {
                  full_name: msg.full_name ?? 'User',
                  initials: msg.initials ?? '?',
                } }
          ))
        }
        const stream = localStream ?? (await acquireMedia())
        const pc = getPeer(msg.user_id)
        addLocalTracksToPeer(pc, stream)
        break
      }

      case 'participant.left': {
        if (!msg.user_id) break
        removePeer(msg.user_id)
        setRemoteSpeaking((prev) => { const n = { ...prev }; delete n[msg.user_id!]; return n })
        setMediaStates((prev) => { const n = { ...prev }; delete n[msg.user_id!]; return n })
        setHandRaisedUsers((prev) => { const s = new Set(prev); s.delete(msg.user_id!); return s })
        setLaserState((prev) => prev?.userId === msg.user_id ? null : prev)
        break
      }

      case 'offer': {
        if (!msg.from || !msg.payload) break
        const pc = getPeer(msg.from)
        await pc.setRemoteDescription(msg.payload as RTCSessionDescriptionInit)
        const answer = await pc.createAnswer()
        await pc.setLocalDescription(answer)
        send({ type: 'answer', to: msg.from, payload: answer })
        break
      }

      case 'answer': {
        if (!msg.from || !msg.payload) break
        const pc = peersRef.current[msg.from]
        if (pc) await pc.setRemoteDescription(msg.payload as RTCSessionDescriptionInit)
        break
      }

      case 'ice_candidate': {
        if (!msg.from || !msg.payload) break
        const pc = peersRef.current[msg.from]
        if (pc) await pc.addIceCandidate(msg.payload as RTCIceCandidateInit)
        break
      }

      case 'speaking_state': {
        if (msg.from) {
          const payload = msg.payload as { speaking: boolean } | undefined
          setRemoteSpeaking((prev) => ({ ...prev, [msg.from!]: payload?.speaking ?? false }))
        }
        break
      }

      // ── In-room controls ──────────────────────────────────────────────────
      case 'room_state': {
        const parts = msg.participants ?? []
        const states: Record<string, MediaState> = {}
        const raised = new Set<string>()
        let activeLaser: LaserState | null = null
        const dirPatch: Record<string, ParticipantDirectoryEntry> = {}
        parts.forEach((p) => {
          states[p.user_id] = { audio: p.audio, video: p.video }
          if (p.hand_raised) raised.add(p.user_id)
          if (p.laser_active) activeLaser = { userId: p.user_id, x: 0, y: 0 }
          if (p.full_name || p.initials) {
            dirPatch[p.user_id] = {
              full_name: p.full_name ?? 'User',
              initials: p.initials ?? '?',
            }
          }
        })
        setMediaStates(states)
        setHandRaisedUsers(raised)
        if (activeLaser) setLaserState(activeLaser)
        if (Object.keys(dirPatch).length > 0) {
          setParticipantDirectory(prev => ({ ...dirPatch, ...prev }))
        }
        break
      }

      case 'media_state': {
        if (!msg.user_id) break
        setMediaStates((prev) => ({
          ...prev,
          [msg.user_id!]: {
            audio: msg.audio ?? prev[msg.user_id!]?.audio ?? true,
            video: msg.video ?? prev[msg.user_id!]?.video ?? true,
          },
        }))
        break
      }

      case 'force_mute': {
        // The server only sends this to the targeted participant.
        if (!isMutedRef.current) {
          const track = localTracksRef.current.audioTrack
          if (track) track.enabled = false
          setIsMuted(true)
          send({ type: 'media_state', audio: false, video: !isCameraOffRef.current })
        }
        break
      }

      case 'emoji_reaction': {
        const userId = msg.from_id ?? msg.user_id ?? msg.from
        if (!userId || !msg.emoji) break
        const id = generateId()
        const reaction: EmojiReaction = { id, userId, emoji: msg.emoji, timestamp: Date.now() }
        setReactionQueue((prev) => {
          const next = [...prev, reaction]
          return next.length > MAX_ACTIVE_REACTIONS ? next.slice(next.length - MAX_ACTIVE_REACTIONS) : next
        })
        setTimeout(() => {
          setReactionQueue((prev) => prev.filter((r) => r.id !== id))
        }, 3000)
        break
      }

      case 'hand_raise': {
        if (!msg.user_id) break
        const raised = msg.raised ?? false
        setHandRaisedUsers((prev) => {
          const s = new Set(prev)
          if (raised) { s.add(msg.user_id!) } else { s.delete(msg.user_id!) }
          return s
        })
        break
      }

      case 'laser_move': {
        if (msg.user_id && msg.x != null && msg.y != null) {
          setLaserState({ userId: msg.user_id, x: msg.x, y: msg.y })
          // If someone else took the laser while we were active, deactivate locally.
          if (isLaserActiveRef.current && msg.user_id !== (meeting?.host_id)) {
            setIsLaserActive(false)
          }
        }
        break
      }

      case 'laser_stop': {
        setLaserState((prev) => prev?.userId === msg.user_id ? null : prev)
        if (msg.user_id && isLaserActiveRef.current) {
          setIsLaserActive(false)
        }
        break
      }

      // ── Chat ──────────────────────────────────────────────────────────────
      case 'chat_message': {
        if (!msg.id || !msg.user_id || msg.body == null) break
        const serverId = msg.id
        const serverClientId = msg.client_id
        const senderId = msg.user_id
        const body = msg.body
        const sentAt = msg.sent_at ? new Date(msg.sent_at).getTime() : Date.now()
        const localUid = localUserIdRef.current

        // Cancel the pending-timeout for this client_id if we have one — the
        // server echo has arrived, so the send is confirmed.
        if (serverClientId) {
          const handle = pendingMsgTimeoutsRef.current[serverClientId]
          if (handle) {
            clearTimeout(handle)
            delete pendingMsgTimeoutsRef.current[serverClientId]
          }
        }

        setMessages(prev => {
          // Dedup: already have this server id → ignore.
          if (prev.some(m => m.id === serverId)) return prev

          // Optimistic reconcile: match by client_id on a pending OR failed
          // entry (failed → retried → confirmed is a valid flow).
          if (serverClientId) {
            const idx = prev.findIndex(m => m.clientId === serverClientId)
            if (idx !== -1) {
              const next = prev.slice()
              next[idx] = {
                ...next[idx],
                id: serverId,
                sentAt,
                pending: false,
                failed: false,
              }
              return next
            }
          }

          // New message from someone else (or own send without matching client_id).
          return [...prev, { id: serverId, userId: senderId, body, sentAt }]
        })

        // Unread: count only remote messages while the panel is closed.
        if (!isChatPanelOpenRef.current && senderId !== localUid) {
          setUnreadCount(c => c + 1)
        }
        break
      }

      // ── Knock flow ────────────────────────────────────────────────────────
      case 'knock.requested': {
        if (msg.knock_id && msg.user_id) {
          setKnocks((prev) => [
            ...prev.filter((k) => k.knock_id !== msg.knock_id),
            { knock_id: msg.knock_id!, user_id: msg.user_id! },
          ])
        }
        break
      }

      case 'knock.resolved':
      case 'knock.cancelled': {
        if (msg.knock_id) setKnocks((prev) => prev.filter((k) => k.knock_id !== msg.knock_id))
        break
      }

      case 'knock.approved': {
        setRoomState('in_meeting')
        await acquireMedia()
        break
      }

      case 'knock.denied':
        setRoomState('denied')
        break

      case 'meeting.ended':
        setRoomState('ended')
        cleanup()
        onEnd?.()
        break

      case 'pong':
        break
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [getPeer, removePeer, send, acquireMedia, addLocalTracksToPeer, localStream, onEnd, meeting])

  // ── Cleanup ───────────────────────────────────────────────────────────────
  const cleanup = useCallback(() => {
    vadRef.current?.stop()
    bgPipelineRef.current?.destroy()
    bgPipelineRef.current = null
    localStream?.getTracks().forEach((t) => t.stop())
    localTracksRef.current = { audioTrack: null, videoTrack: null, screenTrack: null, canvasTrack: null }
    Object.values(peersRef.current).forEach((pc) => pc.close())
    peersRef.current = {}
    // Clear any outstanding pending-message timeouts to avoid state updates
    // after unmount.
    Object.values(pendingMsgTimeoutsRef.current).forEach(clearTimeout)
    pendingMsgTimeoutsRef.current = {}
    wsRef.current?.close()
  }, [localStream])

  // ── Connect effect ────────────────────────────────────────────────────────
  useEffect(() => {
    if (!accessToken || !code) return

    let cancelled = false

    const connect = async () => {
      try {
        const res = await meetingService.getByCode(code)
        if (cancelled) return
        setMeeting(res.data)

        // Populate participant directory from the meeting previews so chat
        // messages render names immediately on first paint.
        if (res.data.participant_previews?.length) {
          setParticipantDirectory(prev => {
            const next = { ...prev }
            for (const p of res.data.participant_previews!) {
              if (!next[p.user_id]) {
                next[p.user_id] = { full_name: p.full_name, initials: p.initials }
              }
            }
            return next
          })
        }

        const wsUrl = meetingService.buildWsUrl(code, accessToken)
        const ws = new WebSocket(wsUrl)
        wsRef.current = ws

        ws.onopen = () => { if (!cancelled) setRoomState('connecting') }

        ws.onmessage = async (e) => {
          try {
            const msg: WsMessage = JSON.parse(e.data as string)
            if (roomState === 'connecting') {
              if (msg.type === 'participant.joined') {
                setRoomState('in_meeting')
                await acquireMedia()
              } else if (msg.type === 'knock.requested') {
                setRoomState('waiting_room')
              }
            }
            await handleMessage(msg)
          } catch { /* ignore parse errors */ }
        }

        ws.onclose = (e) => {
          if (e.code === 4003) setRoomState('denied')
          else if (!cancelled) setRoomState('ended')
          cleanup()
        }

        ws.onerror = () => { if (!cancelled) setRoomState('error') }
      } catch {
        if (!cancelled) setRoomState('error')
      }
    }

    void connect()

    return () => {
      cancelled = true
      cleanup()
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [code, accessToken])

  // ── Chat history fetch on join ────────────────────────────────────────────
  // Runs once each time the user enters `in_meeting` (including re-entry after
  // a reconnect). History is re-fetched so the server is the source of truth;
  // live WS messages dedup by `id` against the fresh list.
  useEffect(() => {
    if (roomState !== 'in_meeting' || !code) return
    let cancelled = false
    ;(async () => {
      setIsLoadingHistory(true)
      setChatHistoryError(null)
      try {
        const { messages: rows, has_more } = await meetingService.listMessages(code, { limit: HISTORY_PAGE_SIZE })
        if (cancelled) return
        setMessages(rows)
        setHasMoreHistory(has_more)
      } catch (err) {
        if (!cancelled) {
          setChatHistoryError(err instanceof Error ? err.message : 'Failed to load chat history')
        }
      } finally {
        if (!cancelled) setIsLoadingHistory(false)
      }
    })()
    return () => { cancelled = true }
  }, [roomState, code])

  // ── Public actions ────────────────────────────────────────────────────────

  const respondToKnock = useCallback((knockId: string, approved: boolean) => {
    send({ type: 'knock.respond', knock_id: knockId, approved })
  }, [send])

  const endMeeting = useCallback(async () => {
    await meetingService.end(code)
    cleanup()
    setRoomState('ended')
    onEnd?.()
  }, [code, cleanup, onEnd])

  const leave = useCallback(() => {
    cleanup()
    setRoomState('ended')
  }, [cleanup])

  // ── Media controls ────────────────────────────────────────────────────────

  const toggleMute = useCallback(() => {
    const track = localTracksRef.current.audioTrack
    if (!track) return
    const nextMuted = !isMutedRef.current
    track.enabled = !nextMuted
    setIsMuted(nextMuted)
    send({ type: 'media_state', audio: !nextMuted, video: !isCameraOffRef.current })
  }, [send])

  const toggleCamera = useCallback(() => {
    const track = localTracksRef.current.videoTrack
    if (!track) return
    const nextOff = !isCameraOffRef.current
    track.enabled = !nextOff
    setIsCameraOff(nextOff)
    send({ type: 'media_state', audio: !isMutedRef.current, video: !nextOff })
    // Pause/resume BG pipeline when camera toggles.
    if (bgPipelineRef.current) {
      nextOff ? bgPipelineRef.current.pause() : bgPipelineRef.current.resume()
    }
  }, [send])

  const shareScreen = useCallback(async () => {
    if (isScreenSharingRef.current) return
    try {
      const screen = await navigator.mediaDevices.getDisplayMedia({ video: true, audio: true })
      const screenTrack = screen.getVideoTracks()[0]
      if (!screenTrack) return

      localTracksRef.current.screenTrack = screenTrack
      setIsScreenSharing(true)
      bgPipelineRef.current?.pause()
      replaceVideoTrack(screenTrack)

      screenTrack.onended = () => stopScreenShare()
    } catch {
      // User dismissed the dialog — ignore.
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [replaceVideoTrack])

  const stopScreenShare = useCallback(() => {
    const track = localTracksRef.current.screenTrack
    track?.stop()
    localTracksRef.current.screenTrack = null
    setIsScreenSharing(false)

    // Resume BG pipeline (or fall back to raw camera).
    if (bgPipelineRef.current && activeBackground.type !== 'none') {
      bgPipelineRef.current.resume()
      const canvasTrack = localTracksRef.current.canvasTrack
      if (canvasTrack) replaceVideoTrack(canvasTrack)
    } else {
      const videoTrack = localTracksRef.current.videoTrack
      if (videoTrack) replaceVideoTrack(videoTrack)
    }
  }, [replaceVideoTrack, activeBackground])

  const forceMute = useCallback((userId: string) => {
    send({ type: 'force_mute', target_id: userId })
  }, [send])

  // ── Hand raise ────────────────────────────────────────────────────────────

  const toggleHand = useCallback(() => {
    const next = !isHandRaisedRef.current
    setIsHandRaised(next)
    send({ type: 'hand_raise', raised: next })
  }, [send])

  // ── Emoji ─────────────────────────────────────────────────────────────────

  const sendEmojiReaction = useCallback((emoji: string) => {
    const now = Date.now()
    if (now - lastEmojiSentRef.current < EMOJI_RATE_LIMIT_MS) return
    lastEmojiSentRef.current = now
    send({ type: 'emoji_reaction', emoji })
  }, [send])

  // ── Laser pointer ─────────────────────────────────────────────────────────

  const toggleLaser = useCallback(() => {
    const next = !isLaserActiveRef.current
    setIsLaserActive(next)
    if (!next) {
      send({ type: 'laser_stop' })
      setLaserState(null)
    }
  }, [send])

  const sendLaserMove = useCallback((x: number, y: number) => {
    const now = Date.now()
    if (now - lastLaserSentRef.current < LASER_THROTTLE_MS) return
    lastLaserSentRef.current = now
    send({ type: 'laser_move', x, y })
  }, [send])

  // ── Virtual background ────────────────────────────────────────────────────

  const uploadCustomBackground = useCallback((file: File): Promise<string | null> => {
    if (!bgSupported) return Promise.resolve('Virtual backgrounds are not supported in this browser')
    if (file.size > CUSTOM_BG_MAX_BYTES) return Promise.resolve('Image must be 2 MB or smaller')

    return new Promise((resolve) => {
      const reader = new FileReader()
      reader.onerror = () => resolve('Failed to read file')
      reader.onload = async (e) => {
        const dataUrl = e.target?.result as string
        if (!dataUrl) { resolve('Failed to read file'); return }

        try {
          localStorage.setItem(CUSTOM_BG_KEY, dataUrl)
          localStorage.setItem(BG_STORAGE_KEY, JSON.stringify({ type: 'image', src: CUSTOM_SRC_SENTINEL, label: 'Custom' }))
        } catch {
          resolve('Not enough storage space for this image')
          return
        }

        setCustomBgSrc(dataUrl)

        const option: BackgroundOption = { type: 'image', src: dataUrl, label: 'Custom' }
        const stream = localStream
        if (stream) {
          if (!bgPipelineRef.current) {
            bgPipelineRef.current = new VirtualBackgroundPipeline(stream)
          }
          try {
            const canvasTrack = await bgPipelineRef.current.setBackground(option)
            if (canvasTrack) {
              localTracksRef.current.canvasTrack = canvasTrack
              if (!isScreenSharingRef.current) replaceVideoTrack(canvasTrack)
            }
          } catch { /* pipeline error — still update UI state */ }
        }

        setActiveBackgroundState(option)
        resolve(null)
      }
      reader.readAsDataURL(file)
    })
  }, [bgSupported, localStream, replaceVideoTrack])

  const setBackground = useCallback(async (option: BackgroundOption) => {
    if (!bgSupported) return

    if (option.type === 'none') {
      bgPipelineRef.current?.destroy()
      bgPipelineRef.current = null
      localTracksRef.current.canvasTrack = null
      setActiveBackgroundState({ type: 'none' })
      localStorage.removeItem(BG_STORAGE_KEY)
      // Restore raw camera track (unless screen sharing).
      if (!isScreenSharingRef.current) {
        const videoTrack = localTracksRef.current.videoTrack
        if (videoTrack) replaceVideoTrack(videoTrack)
      }
      return
    }

    const stream = localStream
    if (!stream) return

    if (!bgPipelineRef.current) {
      bgPipelineRef.current = new VirtualBackgroundPipeline(stream)
    }

    const canvasTrack = await bgPipelineRef.current.setBackground(option)
    if (canvasTrack) {
      localTracksRef.current.canvasTrack = canvasTrack
      if (!isScreenSharingRef.current) {
        replaceVideoTrack(canvasTrack)
      }
    }

    setActiveBackgroundState(option)
    try { localStorage.setItem(BG_STORAGE_KEY, JSON.stringify(option)) } catch { /* ignore */ }
  }, [bgSupported, localStream, replaceVideoTrack])

  // ── Chat ──────────────────────────────────────────────────────────────────

  const fetchHistory = useCallback(async () => {
    if (!code) return
    setIsLoadingHistory(true)
    setChatHistoryError(null)
    try {
      const { messages: rows, has_more } = await meetingService.listMessages(code, { limit: HISTORY_PAGE_SIZE })
      setMessages(rows)
      setHasMoreHistory(has_more)
    } catch (err) {
      setChatHistoryError(err instanceof Error ? err.message : 'Failed to load chat history')
    } finally {
      setIsLoadingHistory(false)
    }
  }, [code])

  const retryHistoryFetch = useCallback(() => { void fetchHistory() }, [fetchHistory])

  const openChatPanel = useCallback(() => {
    setIsChatPanelOpen(true)
    setUnreadCount(0)
  }, [])

  const closeChatPanel = useCallback(() => {
    setIsChatPanelOpen(false)
  }, [])

  const dismissChatSendError = useCallback(() => { setChatSendError(null) }, [])

  // schedulePendingTimeout marks a pending entry `failed` if no echo arrives
  // within PENDING_MESSAGE_TIMEOUT_MS. Also exposes a way for the UI to
  // trigger a retry.
  const schedulePendingTimeout = useCallback((clientId: string) => {
    const handle = setTimeout(() => {
      delete pendingMsgTimeoutsRef.current[clientId]
      setMessages(prev => prev.map(m => (
        m.clientId === clientId && m.pending
          ? { ...m, pending: false, failed: true }
          : m
      )))
    }, PENDING_MESSAGE_TIMEOUT_MS)
    pendingMsgTimeoutsRef.current[clientId] = handle
  }, [])

  const sendChatMessage = useCallback((rawBody: string) => {
    const body = rawBody.trim()
    if (!body) return

    if (body.length > MAX_MESSAGE_LENGTH) {
      setChatSendError(`Messages are limited to ${MAX_MESSAGE_LENGTH} characters`)
      return
    }

    if (wsRef.current?.readyState !== WebSocket.OPEN) {
      setChatSendError('Not connected — message could not be sent')
      return
    }

    // Rolling rate limit: last RATE_LIMIT_MESSAGES timestamps within window.
    const now = Date.now()
    const recent = chatSendTimestampsRef.current.filter(t => now - t < RATE_LIMIT_WINDOW_MS)
    if (recent.length >= RATE_LIMIT_MESSAGES) {
      chatSendTimestampsRef.current = recent
      setChatFlashKey(k => k + 1)
      return
    }
    recent.push(now)
    chatSendTimestampsRef.current = recent

    const uid = localUserIdRef.current
    if (!uid) return

    const clientId = generateId()
    send({ type: 'chat_message', client_id: clientId, body })
    schedulePendingTimeout(clientId)

    setMessages(prev => [
      ...prev,
      { id: clientId, clientId, userId: uid, body, sentAt: now, pending: true },
    ])
  }, [send, schedulePendingTimeout])

  // Retry a previously-failed local message by re-sending with the SAME
  // client_id. The backend tolerates duplicate client_ids (each insert gets
  // its own server id) but on our side we re-use the local id so React keys
  // stay stable and the echoed message reconciles in-place.
  const retrySendMessage = useCallback((localId: string) => {
    const entry = messages.find(m => m.id === localId || m.clientId === localId)
    if (!entry || !entry.clientId) return
    if (!entry.failed && !entry.pending) return

    if (wsRef.current?.readyState !== WebSocket.OPEN) {
      setChatSendError('Not connected — message could not be sent')
      return
    }

    // Mark pending again and reschedule timeout.
    setMessages(prev => prev.map(m => (
      m.id === entry.id ? { ...m, pending: true, failed: false } : m
    )))
    send({ type: 'chat_message', client_id: entry.clientId, body: entry.body })
    schedulePendingTimeout(entry.clientId)
  }, [messages, send, schedulePendingTimeout])

  const deleteFailedMessage = useCallback((localId: string) => {
    setMessages(prev => prev.filter(m => m.id !== localId))
  }, [])

  const loadOlderMessages = useCallback(async () => {
    if (!code || isLoadingHistory || !hasMoreHistory || messages.length === 0) return
    setIsLoadingHistory(true)
    try {
      const oldest = messages[0]
      const before = new Date(oldest.sentAt).toISOString()
      const { messages: older, has_more } = await meetingService.listMessages(code, {
        before,
        limit: HISTORY_PAGE_SIZE,
      })
      // Dedup by id before prepending (the oldest of the current list may
      // match the most recent of the older page in theory; we trust `before`
      // exclusivity but keep the safety check).
      setMessages(prev => {
        const existingIds = new Set(prev.map(m => m.id))
        const unique = older.filter(m => !existingIds.has(m.id))
        return [...unique, ...prev]
      })
      setHasMoreHistory(has_more)
    } catch (err) {
      setChatHistoryError(err instanceof Error ? err.message : 'Failed to load more messages')
    } finally {
      setIsLoadingHistory(false)
    }
  }, [code, isLoadingHistory, hasMoreHistory, messages])

  // ─────────────────────────────────────────────────────────────────────────
  return {
    meeting,
    roomState,
    isSpeaking,
    audioLevel,
    remoteSpeaking,
    knocks,
    localStream,
    peers,
    respondToKnock,
    endMeeting,
    leave,
    isMuted,
    isCameraOff,
    isScreenSharing,
    toggleMute,
    toggleCamera,
    shareScreen,
    stopScreenShare,
    forceMute,
    isHandRaised,
    handRaisedUsers,
    handRaisedCount: handRaisedUsers.size,
    toggleHand,
    reactionQueue,
    sendEmojiReaction,
    laserState,
    isLaserActive,
    toggleLaser,
    sendLaserMove,
    activeBackground,
    bgSupported,
    setBackground,
    customBgSrc,
    uploadCustomBackground,
    mediaStates,
    // ── chat ──────────────────────────────────────────────────────────────
    messages,
    unreadCount,
    isChatPanelOpen,
    isLoadingHistory,
    hasMoreHistory,
    chatHistoryError,
    participantDirectory,
    chatFlashKey,
    chatSendError,
    openChatPanel,
    closeChatPanel,
    sendChatMessage,
    retrySendMessage,
    deleteFailedMessage,
    loadOlderMessages,
    retryHistoryFetch,
    dismissChatSendError,
  }
}
