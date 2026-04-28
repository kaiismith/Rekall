import { useEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  Alert,
  Avatar,
  Box,
  Button,
  Chip,
  CircularProgress,
  Divider,
  IconButton,
  Paper,
  Stack,
  Tooltip,
  Typography,
} from '@mui/material'
import CallEndIcon from '@mui/icons-material/CallEnd'
import CheckIcon from '@mui/icons-material/Check'
import CloseIcon from '@mui/icons-material/Close'
import MicOffIcon from '@mui/icons-material/MicOff'
import MicIcon from '@mui/icons-material/Mic'
import VideocamOffIcon from '@mui/icons-material/VideocamOff'
import VideocamIcon from '@mui/icons-material/Videocam'
import BackHandIcon from '@mui/icons-material/BackHand'
import SettingsIcon from '@mui/icons-material/SettingsOutlined'
import { useMeeting } from '@/hooks/useMeeting'
import { useAuthStore } from '@/store/authStore'
import { MicButton } from '@/components/meeting/MicButton'
import { CameraButton } from '@/components/meeting/CameraButton'
import { ShareButton } from '@/components/meeting/ShareButton'
import { HandButton } from '@/components/meeting/HandButton'
import { EmojiButton } from '@/components/meeting/EmojiButton'
import { BackgroundButton } from '@/components/meeting/BackgroundButton'
import { ChatButton } from '@/components/meeting/ChatButton'
import { ChatPanel } from '@/components/meeting/chat/ChatPanel'
import { CaptionsButton } from '@/components/meeting/CaptionsButton'
import { MeetingCaptionsPanel } from '@/components/meeting/MeetingCaptionsPanel'
import { useASR } from '@/hooks/useASR'
import { DeviceSettingsDialog } from '@/components/meeting/DeviceSettingsDialog'
import { tokens } from '@/theme'
import type { EmojiReaction } from '@/types/meeting'

// ─── Emoji float animation injected once ─────────────────────────────────────
const EMOJI_STYLE_ID = 'rekall-emoji-float'
if (typeof document !== 'undefined' && !document.getElementById(EMOJI_STYLE_ID)) {
  const style = document.createElement('style')
  style.id = EMOJI_STYLE_ID
  style.textContent = `
    @keyframes rekall-float-fade {
      0%   { transform: translateY(0) scale(1);    opacity: 1; }
      100% { transform: translateY(-80px) scale(1.4); opacity: 0; }
    }
    .rekall-emoji-float {
      animation: rekall-float-fade 3s ease-out forwards;
      position: absolute;
      font-size: 2rem;
      pointer-events: none;
      bottom: 48px;
      transform-origin: center bottom;
    }
  `
  document.head.appendChild(style)
}

export function MeetingRoomPage() {
  const { code } = useParams<{ code: string }>()
  const navigate = useNavigate()
  const { user } = useAuthStore()
  const localVideoRef = useRef<HTMLVideoElement>(null)
  const videoGridRef = useRef<HTMLDivElement>(null)
  const [settingsOpen, setSettingsOpen] = useState(false)

  const {
    meeting,
    roomState,
    isSpeaking,
    audioLevel,
    remoteSpeaking,
    knocks,
    localStream,
    localPreviewStream,
    peers,
    respondToKnock,
    joinNow,
    availableDevices,
    selectedDeviceIds,
    switchCamera,
    switchMic,
    switchSpeaker,
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
    handRaisedCount,
    toggleHand,
    reactionQueue,
    sendEmojiReaction,
    activeBackground,
    bgSupported,
    setBackground,
    customBgSrc,
    uploadCustomBackground,
    mediaStates,
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
    captions,
    sendCaptionChunk,
    sendCaptionFinal,
  } = useMeeting({
    code: code ?? '',
    onEnd: () => {
      console.warn('%c[meeting] page: onEnd → /meetings', 'color:#f59e0b;font-weight:600')
      navigate('/meetings')
    },
  })

  // ── Live captions: per-user opt-in (no host permission, no meeting flag).
  // Each participant decides for themselves whether to open the captions
  // panel. Turning it on starts a local ASR session against your own mic
  // and broadcasts the resulting chunks via the meeting WS hub; turning it
  // off tears the session down. Other participants who also have captions
  // on will see your text attributed to you, and vice versa.
  const [captionsOn, setCaptionsOn] = useState(false)

  // sessionId is set by useASR once the server issues a token. Held in a ref
  // so the inline callbacks below don't capture a stale value across renders.
  const asrSessionIdRef = useRef<string | null>(null)

  const asr = useASR(code ?? null, 'meeting', {
    onPartial: (text, segId) => sendCaptionChunk('partial', text, segId),
    onFinalSegment: (event) => {
      const sid = asrSessionIdRef.current
      if (sid) {
        // Full-payload path: backend hub broadcasts AND persists.
        sendCaptionFinal(event, sid)
      } else {
        // Defensive fallback: relay only (no persistence). Should not
        // normally hit because the session_id is set before any final.
        sendCaptionChunk('final', event.text, String(event.segment_id))
      }
    },
  })
  asrSessionIdRef.current = asr.sessionId

  // Stable handle to the latest asr API so the intent-driven effect below
  // doesn't re-fire on every render (asr is a fresh object identity each
  // render). useASR owns its own reconnect backoff for transient drops; the
  // page only signals user intent.
  const asrRef = useRef(asr)
  asrRef.current = asr

  // Drive ASR purely from user intent. Starting/stopping here in response to
  // asr.state changes caused a feedback loop: a session that ended (for any
  // reason — flush, server idle, network blip) flipped state back to
  // 'ended', this effect re-fired, called start() again, repeat — hundreds
  // of sessions per second. The fix is to react ONLY to the inputs the user
  // controls.
  useEffect(() => {
    const shouldStream = captionsOn && roomState === 'in_meeting' && !isMuted
    if (shouldStream) {
      void asrRef.current.start()
    } else {
      void asrRef.current.stop()
    }
  }, [captionsOn, roomState, isMuted])

  // Log every roomState transition so we can see in devtools exactly where
  // the page lands. Easy to grep on `[meeting]`.
  useEffect(() => {
    console.warn('%c[meeting] page: roomState =', 'color:#a78bfa;font-weight:600', roomState)
  }, [roomState])

  // Attach the local PREVIEW stream (mirrors the active outbound video track:
  // screen share > virtual-bg canvas > raw camera) to the local <video> tile,
  // so the user sees what their peers see. Uses a callback ref via the existing
  // localVideoRef so the binding fires both on stream change AND on element
  // mount — required when transitioning from the device-check screen, where
  // a different <video> instance was used and the new one comes up empty.
  const stream = localPreviewStream ?? localStream
  useEffect(() => {
    const el = localVideoRef.current
    if (el && stream && el.srcObject !== stream) {
      el.srcObject = stream
    }
  }, [stream, isCameraOff, roomState])

  // ── Device check / Connecting / waiting / denied / ended ──────────────────

  // Don't render the device-check screen until the meeting metadata has
  // loaded — otherwise users hitting an ended meeting see the preview UI
  // pop up for a frame before flipping to "Meeting Ended". Show a quiet
  // loading spinner instead while the initial getByCode is in flight.
  if (roomState === 'device_check' && !meeting) {
    return (
      <Box
        display="flex"
        flexDirection="column"
        alignItems="center"
        justifyContent="center"
        height="100vh"
        gap={2}
      >
        <CircularProgress />
      </Box>
    )
  }

  if (roomState === 'device_check') {
    return (
      <>
        <DeviceCheckScreen
          meetingTitle={meeting?.title || 'Meeting'}
          meetingCode={meeting?.code ?? ''}
          previewStream={localPreviewStream ?? localStream}
          isMuted={isMuted}
          isCameraOff={isCameraOff}
          audioLevel={audioLevel}
          onToggleMic={toggleMute}
          onToggleCamera={toggleCamera}
          onJoin={joinNow}
          onCancel={() => navigate('/meetings')}
          onOpenSettings={() => setSettingsOpen(true)}
        />
        <DeviceSettingsDialog
          open={settingsOpen}
          onClose={() => setSettingsOpen(false)}
          cameras={availableDevices.cameras}
          mics={availableDevices.mics}
          speakers={availableDevices.speakers}
          selectedCameraId={selectedDeviceIds.camera}
          selectedMicId={selectedDeviceIds.mic}
          selectedSpeakerId={selectedDeviceIds.speaker}
          onSwitchCamera={switchCamera}
          onSwitchMic={switchMic}
          onSwitchSpeaker={switchSpeaker}
        />
      </>
    )
  }

  if (roomState === 'connecting') {
    return (
      <Box
        display="flex"
        flexDirection="column"
        alignItems="center"
        justifyContent="center"
        height="100vh"
        gap={2}
      >
        <CircularProgress />
        <Typography color="text.secondary">Connecting…</Typography>
      </Box>
    )
  }

  if (roomState === 'waiting_room') {
    return (
      <Box
        display="flex"
        flexDirection="column"
        alignItems="center"
        justifyContent="center"
        height="100vh"
        gap={3}
      >
        <Typography variant="h5">Waiting for admission</Typography>
        <Typography color="text.secondary">A participant will admit you shortly.</Typography>
        <CircularProgress size={28} />
      </Box>
    )
  }

  if (roomState === 'denied') {
    return (
      <Box
        display="flex"
        flexDirection="column"
        alignItems="center"
        justifyContent="center"
        height="100vh"
        gap={2}
      >
        <Typography variant="h5" color="error">
          Access Denied
        </Typography>
        <Typography color="text.secondary">Your request to join was declined.</Typography>
        <Button variant="outlined" onClick={() => navigate('/meetings')}>
          Back to Meetings
        </Button>
      </Box>
    )
  }

  if (roomState === 'ended' || roomState === 'error') {
    return (
      <Box
        display="flex"
        flexDirection="column"
        alignItems="center"
        justifyContent="center"
        height="100vh"
        gap={2}
      >
        <Typography variant="h5">Meeting Ended</Typography>
        <Button variant="outlined" onClick={() => navigate('/meetings')}>
          Back to Meetings
        </Button>
      </Box>
    )
  }

  // ── In-meeting UI ─────────────────────────────────────────────────────────

  const isHost = meeting?.host_id === user?.id
  const peerIds = Object.keys(peers)

  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        height: '100vh',
        bgcolor: 'background.default',
      }}
    >
      {/* ── Top bar ─────────────────────────────────────────────────────────── */}
      <Box
        sx={{
          px: 3,
          py: 1.5,
          borderBottom: '1px solid',
          borderColor: 'divider',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          flexShrink: 0,
        }}
      >
        <Stack direction="row" spacing={1} alignItems="center">
          <Typography variant="h6" fontWeight={600}>
            {meeting?.title || 'Meeting'}
          </Typography>
          <Chip label={meeting?.code} size="small" sx={{ fontFamily: tokens.fonts.mono }} />
          <Chip label={meeting?.type} size="small" variant="outlined" />
          <Chip label={`${peerIds.length + 1} participants`} size="small" variant="outlined" />
          {handRaisedCount > 0 && (
            <Chip
              icon={<BackHandIcon sx={{ fontSize: '0.9rem !important' }} />}
              label={handRaisedCount}
              size="small"
              color="warning"
            />
          )}
        </Stack>

        <Stack direction="row" spacing={1}>
          {isHost ? (
            <Button
              variant="contained"
              color="error"
              startIcon={<CallEndIcon />}
              onClick={endMeeting}
              size="small"
            >
              End Meeting
            </Button>
          ) : (
            <Button
              variant="outlined"
              color="error"
              startIcon={<CallEndIcon />}
              onClick={leave}
              size="small"
            >
              Leave
            </Button>
          )}
        </Stack>
      </Box>

      {/* ── Screen share banner ──────────────────────────────────────────────── */}
      {isScreenSharing && (
        <Alert severity="info" sx={{ borderRadius: 0, py: 0.5 }}>
          You are sharing your screen
        </Alert>
      )}

      {/* ── Main area: grid + knock sidebar ─────────────────────────────────── */}
      <Box sx={{ display: 'flex', flex: 1, overflow: 'hidden' }}>
        {/* ── Video grid ───────────────────────────────────────────────────── */}
        <Box
          ref={videoGridRef}
          sx={{
            flex: 1,
            position: 'relative',
            p: 2,
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))',
            gap: 2,
            alignContent: 'start',
            overflow: 'auto',
          }}
        >
          {/* Local tile */}
          <VideoTile
            label={isScreenSharing ? 'You (sharing)' : 'You'}
            isMuted={isMuted}
            isSpeaking={isSpeaking}
            audioLevel={audioLevel}
            isCameraOff={isCameraOff}
            isHandRaised={isHandRaised}
            reactions={reactionQueue.filter((r) => r.userId === user?.id)}
            isLocal
          >
            {/* Always mount the <video> element — toggling it via the parent
                conditional unmounted/remounted the node and the new instance
                started without srcObject (the attach effect doesn't re-run if
                localStream hasn't changed). Hide it via CSS instead so the
                MediaStream binding survives camera on→off→on cycles. */}
            <video
              ref={localVideoRef}
              autoPlay
              muted
              playsInline
              style={{
                width: '100%',
                height: '100%',
                objectFit: 'cover',
                // Mirror only the camera preview — screen-share content
                // would read backwards if mirrored.
                transform: isScreenSharing ? 'none' : 'scaleX(-1)',
                display: isCameraOff && !isScreenSharing ? 'none' : 'block',
              }}
            />
          </VideoTile>

          {/* Remote tiles */}
          {peerIds.map((uid) => {
            const state = mediaStates[uid]
            const speaking = remoteSpeaking[uid] ?? false
            const handUp = handRaisedUsers.has(uid)
            const camOff = state?.video === false
            const micOff = state?.audio === false

            return (
              <VideoTile
                key={uid}
                label={uid.slice(0, 8) + '…'}
                isMuted={micOff}
                isSpeaking={speaking}
                audioLevel={0}
                isCameraOff={camOff}
                isHandRaised={handUp}
                reactions={reactionQueue.filter((r) => r.userId === uid)}
                hostAction={isHost ? () => forceMute(uid) : undefined}
              >
                {!camOff && <RemoteVideo peerConnection={peers[uid]} />}
              </VideoTile>
            )
          })}
        </Box>

        {/* ── Chat panel ──────────────────────────────────────────────────── */}
        <ChatPanel
          isOpen={isChatPanelOpen}
          onClose={closeChatPanel}
          messages={messages}
          directory={participantDirectory}
          localUserId={user?.id ?? null}
          hasMore={hasMoreHistory}
          isLoading={isLoadingHistory}
          historyError={chatHistoryError}
          sendError={chatSendError}
          onLoadOlder={loadOlderMessages}
          onRetry={retryHistoryFetch}
          onSend={sendChatMessage}
          onRetrySend={retrySendMessage}
          onDeleteFailed={deleteFailedMessage}
          onDismissSendError={dismissChatSendError}
          composerDisabled={roomState !== 'in_meeting'}
          flashKey={chatFlashKey}
        />

        {/* ── Knock sidebar ────────────────────────────────────────────────── */}
        {knocks.length > 0 && (
          <Box
            sx={{
              width: 280,
              borderLeft: '1px solid',
              borderColor: 'divider',
              p: 2,
              overflowY: 'auto',
              flexShrink: 0,
            }}
          >
            <Typography variant="subtitle2" fontWeight={600} mb={1}>
              Waiting to Join ({knocks.length})
            </Typography>
            <Divider sx={{ mb: 1 }} />
            <Stack spacing={1}>
              {knocks.map((knock) => (
                <Paper key={knock.knock_id} sx={{ p: 1.5 }}>
                  <Stack direction="row" spacing={1} alignItems="center">
                    <Avatar sx={{ width: 32, height: 32, fontSize: '0.8rem' }}>
                      {knock.user_id.slice(0, 2).toUpperCase()}
                    </Avatar>
                    <Typography variant="body2" sx={{ flex: 1 }} noWrap>
                      {knock.user_id.slice(0, 12)}…
                    </Typography>
                  </Stack>
                  <Stack direction="row" spacing={0.5} mt={1}>
                    <Button
                      size="small"
                      variant="contained"
                      color="success"
                      startIcon={<CheckIcon />}
                      fullWidth
                      onClick={() => respondToKnock(knock.knock_id, true)}
                    >
                      Admit
                    </Button>
                    <Button
                      size="small"
                      variant="outlined"
                      color="error"
                      startIcon={<CloseIcon />}
                      fullWidth
                      onClick={() => respondToKnock(knock.knock_id, false)}
                    >
                      Deny
                    </Button>
                  </Stack>
                </Paper>
              ))}
            </Stack>
          </Box>
        )}

        {/* ── Live-captions sidebar ────────────────────────────────────────── */}
        {/* Personal opt-in: shown only to participants who clicked the
            captions button in their own control bar. The captions array is
            maintained even when the panel is closed, so reopening shows
            recent history immediately. */}
        {captionsOn && roomState === 'in_meeting' && (
          <Box
            sx={{
              width: 340,
              borderLeft: '1px solid',
              borderColor: 'divider',
              p: 2,
              flexShrink: 0,
              display: 'flex',
            }}
          >
            <MeetingCaptionsPanel
              captions={captions}
              directory={participantDirectory}
              localUserId={user?.id ?? null}
              isStreamingLocal={asr.state === 'streaming'}
              isDisabled={false}
            />
          </Box>
        )}
      </Box>

      {/* ── Control bar ─────────────────────────────────────────────────────── */}
      <Box
        sx={{
          borderTop: '1px solid',
          borderColor: 'divider',
          px: 3,
          py: 1.5,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          gap: 1.5,
          flexShrink: 0,
          bgcolor: 'background.paper',
        }}
      >
        <MicButton isMuted={isMuted} audioLevel={audioLevel} onToggle={toggleMute} />
        <CameraButton isCameraOff={isCameraOff} onToggle={toggleCamera} />
        <ShareButton
          isScreenSharing={isScreenSharing}
          onShare={shareScreen}
          onStop={stopScreenShare}
        />
        <HandButton isHandRaised={isHandRaised} onToggle={toggleHand} />
        <EmojiButton onSend={sendEmojiReaction} />
        <BackgroundButton
          active={activeBackground}
          onSelect={setBackground}
          onUpload={uploadCustomBackground}
          customBgSrc={customBgSrc}
          disabled={!bgSupported}
        />
        <ChatButton
          unreadCount={unreadCount}
          isOpen={isChatPanelOpen}
          onToggle={isChatPanelOpen ? closeChatPanel : openChatPanel}
        />
        <CaptionsButton enabled={captionsOn} onToggle={() => setCaptionsOn((on) => !on)} />
        <Tooltip title="Device settings">
          <IconButton
            size="small"
            onClick={() => setSettingsOpen(true)}
            sx={{
              width: 44,
              height: 44,
              bgcolor: 'rgba(255,255,255,0.04)',
              color: 'text.secondary',
              border: '1px solid rgba(255,255,255,0.06)',
              '&:hover': { bgcolor: 'rgba(255,255,255,0.08)', color: 'text.primary' },
            }}
            aria-label="Device settings"
          >
            <SettingsIcon fontSize="small" />
          </IconButton>
        </Tooltip>
      </Box>

      <DeviceSettingsDialog
        open={settingsOpen}
        onClose={() => setSettingsOpen(false)}
        cameras={availableDevices.cameras}
        mics={availableDevices.mics}
        speakers={availableDevices.speakers}
        selectedCameraId={selectedDeviceIds.camera}
        selectedMicId={selectedDeviceIds.mic}
        selectedSpeakerId={selectedDeviceIds.speaker}
        onSwitchCamera={switchCamera}
        onSwitchMic={switchMic}
        onSwitchSpeaker={switchSpeaker}
      />
    </Box>
  )
}

// ─── VideoTile ────────────────────────────────────────────────────────────────

interface VideoTileProps {
  label: string
  isMuted: boolean
  isSpeaking: boolean
  audioLevel: number
  isCameraOff: boolean
  isHandRaised: boolean
  reactions: EmojiReaction[]
  isLocal?: boolean
  hostAction?: () => void // mute-other button shown to host
  children?: React.ReactNode
}

function VideoTile({
  label,
  isMuted,
  isSpeaking,
  audioLevel,
  isCameraOff,
  isHandRaised,
  reactions,
  isLocal,
  hostAction,
  children,
}: VideoTileProps) {
  // Border width scales 2–4 px with audio level for local tile; binary for remote.
  const borderWidth = isLocal ? `${2 + audioLevel * 2}px` : '2px'

  return (
    <Paper
      sx={{
        position: 'relative',
        aspectRatio: '16/9',
        overflow: 'hidden',
        border: isSpeaking ? `${borderWidth} solid` : '2px solid transparent',
        borderColor: isSpeaking ? 'primary.main' : 'transparent',
        transition: 'border-color 0.1s ease, border-width 0.06s ease-out',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        bgcolor: 'background.paper',
      }}
    >
      {/* Video or avatar */}
      {isCameraOff ? (
        <Avatar sx={{ width: 64, height: 64, fontSize: '1.5rem', bgcolor: 'primary.dark' }}>
          {label.slice(0, 2).toUpperCase()}
        </Avatar>
      ) : (
        children
      )}

      {/* Mute icon — bottom left */}
      <Box
        sx={{
          position: 'absolute',
          bottom: 8,
          left: 8,
          display: 'flex',
          alignItems: 'center',
          gap: 0.5,
        }}
      >
        {isMuted ? (
          <MicOffIcon fontSize="small" sx={{ color: 'error.light' }} />
        ) : (
          isSpeaking && <MicIcon fontSize="small" color="primary" />
        )}
        <Typography
          variant="caption"
          sx={{ bgcolor: 'rgba(0,0,0,0.55)', px: 0.5, borderRadius: 0.5 }}
        >
          {label}
        </Typography>
      </Box>

      {/* Hand raise — top right */}
      {isHandRaised && (
        <Box sx={{ position: 'absolute', top: 8, right: 8 }}>
          <Typography fontSize="1.2rem" lineHeight={1}>
            ✋
          </Typography>
        </Box>
      )}

      {/* Host mute button — top right (on hover) */}
      {hostAction && (
        <Tooltip title="Mute participant">
          <IconButton
            size="small"
            onClick={hostAction}
            sx={{
              position: 'absolute',
              top: 8,
              right: isHandRaised ? 40 : 8,
              bgcolor: 'rgba(0,0,0,0.5)',
              color: 'white',
              opacity: 0,
              '.MuiPaper-root:hover &': { opacity: 1 },
              transition: 'opacity 0.15s',
            }}
          >
            <MicOffIcon fontSize="small" />
          </IconButton>
        </Tooltip>
      )}

      {/* Camera-off indicator */}
      {isCameraOff && (
        <Box sx={{ position: 'absolute', bottom: 8, right: 8 }}>
          <VideocamOffIcon fontSize="small" sx={{ color: 'text.secondary' }} />
        </Box>
      )}

      {/* Floating emoji reactions */}
      {reactions.map((r, i) => (
        <span
          key={r.id}
          className="rekall-emoji-float"
          style={{ left: `calc(50% + ${((i % 5) - 2) * 18}px)` }}
        >
          {r.emoji}
        </span>
      ))}
    </Paper>
  )
}

// ─── RemoteVideo ──────────────────────────────────────────────────────────────

function RemoteVideo({ peerConnection }: { peerConnection: RTCPeerConnection }) {
  const videoRef = useRef<HTMLVideoElement>(null)

  useEffect(() => {
    const handler = (e: RTCTrackEvent) => {
      if (videoRef.current) videoRef.current.srcObject = e.streams[0] ?? null
    }
    peerConnection.addEventListener('track', handler)
    return () => peerConnection.removeEventListener('track', handler)
  }, [peerConnection])

  return (
    <video
      ref={videoRef}
      autoPlay
      playsInline
      style={{ width: '100%', height: '100%', objectFit: 'cover' }}
    />
  )
}

// ─── DeviceCheckScreen ────────────────────────────────────────────────────────

interface DeviceCheckScreenProps {
  meetingTitle: string
  meetingCode: string
  previewStream: MediaStream | null
  isMuted: boolean
  isCameraOff: boolean
  audioLevel: number
  onToggleMic: () => void
  onToggleCamera: () => void
  onJoin: () => void
  onCancel: () => void
  onOpenSettings: () => void
}

function DeviceCheckScreen({
  meetingTitle,
  meetingCode,
  previewStream,
  isMuted,
  isCameraOff,
  audioLevel,
  onToggleMic,
  onToggleCamera,
  onJoin,
  onCancel,
  onOpenSettings,
}: DeviceCheckScreenProps) {
  const previewVideoRef = useRef<HTMLVideoElement>(null)

  useEffect(() => {
    const el = previewVideoRef.current
    if (el && previewStream && el.srcObject !== previewStream) {
      el.srcObject = previewStream
    }
  }, [previewStream])

  const meterPct = Math.min(100, Math.round(audioLevel * 100))

  return (
    <Box
      sx={{
        minHeight: '100vh',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        bgcolor: 'background.default',
        px: { xs: 2, sm: 3 },
        py: { xs: 4, sm: 6 },
      }}
    >
      <Stack spacing={3} sx={{ width: '100%', maxWidth: 720, alignItems: 'center' }}>
        <Box sx={{ textAlign: 'center' }}>
          <Typography
            variant="overline"
            sx={{
              color: 'text.secondary',
              fontWeight: 700,
              letterSpacing: '0.12em',
              display: 'block',
              mb: 0.5,
            }}
          >
            Ready to join
          </Typography>
          <Typography
            component="h1"
            sx={{
              fontWeight: 700,
              letterSpacing: '-0.02em',
              fontSize: { xs: '1.5rem', sm: '1.875rem' },
              color: 'text.primary',
            }}
          >
            {meetingTitle}
          </Typography>
          {meetingCode && (
            <Typography
              variant="caption"
              sx={{
                color: 'text.secondary',
                fontFamily: tokens.fonts.mono,
                mt: 0.5,
                display: 'block',
              }}
            >
              {meetingCode}
            </Typography>
          )}
        </Box>

        {/* Camera preview */}
        <Paper
          sx={{
            position: 'relative',
            width: '100%',
            aspectRatio: '16 / 9',
            overflow: 'hidden',
            borderRadius: '14px',
            bgcolor: '#0a0b12',
            border: '1px solid rgba(255,255,255,0.06)',
            boxShadow: '0 12px 40px rgba(0,0,0,0.45)',
          }}
        >
          {previewStream && (
            <video
              ref={previewVideoRef}
              autoPlay
              muted
              playsInline
              style={{
                width: '100%',
                height: '100%',
                objectFit: 'cover',
                transform: 'scaleX(-1)',
                display: isCameraOff ? 'none' : 'block',
              }}
            />
          )}

          {isCameraOff && (
            <Stack
              sx={{
                position: 'absolute',
                inset: 0,
                alignItems: 'center',
                justifyContent: 'center',
                gap: 1.5,
              }}
            >
              <Box
                sx={{
                  width: 64,
                  height: 64,
                  borderRadius: '50%',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  bgcolor: 'rgba(255,255,255,0.05)',
                  color: 'text.secondary',
                }}
              >
                <VideocamOffIcon sx={{ fontSize: '2rem' }} />
              </Box>
              <Typography variant="body2" color="text.secondary">
                Camera is off
              </Typography>
            </Stack>
          )}

          {/* Bottom controls overlay */}
          <Stack
            direction="row"
            spacing={1.5}
            sx={{
              position: 'absolute',
              bottom: 16,
              left: '50%',
              transform: 'translateX(-50%)',
              p: 1,
              borderRadius: '999px',
              bgcolor: 'rgba(0,0,0,0.55)',
              backdropFilter: 'blur(8px)',
              border: '1px solid rgba(255,255,255,0.08)',
            }}
          >
            <Tooltip title={isMuted ? 'Turn on microphone' : 'Mute microphone'}>
              <IconButton
                onClick={onToggleMic}
                sx={{
                  bgcolor: isMuted ? 'rgba(239,68,68,0.18)' : 'rgba(255,255,255,0.06)',
                  color: isMuted ? '#fca5a5' : 'text.primary',
                  border: '1px solid',
                  borderColor: isMuted ? 'rgba(239,68,68,0.3)' : 'rgba(255,255,255,0.08)',
                  '&:hover': {
                    bgcolor: isMuted ? 'rgba(239,68,68,0.24)' : 'rgba(255,255,255,0.1)',
                  },
                }}
              >
                {isMuted ? <MicOffIcon /> : <MicIcon />}
              </IconButton>
            </Tooltip>

            <Tooltip title={isCameraOff ? 'Turn on camera' : 'Turn off camera'}>
              <IconButton
                onClick={onToggleCamera}
                sx={{
                  bgcolor: isCameraOff ? 'rgba(239,68,68,0.18)' : 'rgba(255,255,255,0.06)',
                  color: isCameraOff ? '#fca5a5' : 'text.primary',
                  border: '1px solid',
                  borderColor: isCameraOff ? 'rgba(239,68,68,0.3)' : 'rgba(255,255,255,0.08)',
                  '&:hover': {
                    bgcolor: isCameraOff ? 'rgba(239,68,68,0.24)' : 'rgba(255,255,255,0.1)',
                  },
                }}
              >
                {isCameraOff ? <VideocamOffIcon /> : <VideocamIcon />}
              </IconButton>
            </Tooltip>

            <Tooltip title="Device settings">
              <IconButton
                onClick={onOpenSettings}
                aria-label="Device settings"
                sx={{
                  bgcolor: 'rgba(255,255,255,0.06)',
                  color: 'text.primary',
                  border: '1px solid rgba(255,255,255,0.08)',
                  '&:hover': { bgcolor: 'rgba(255,255,255,0.1)' },
                }}
              >
                <SettingsIcon />
              </IconButton>
            </Tooltip>
          </Stack>
        </Paper>

        {/* Mic-level meter (only when mic is on) */}
        {!isMuted && (
          <Box sx={{ width: '100%', maxWidth: 320 }}>
            <Stack direction="row" spacing={1} alignItems="center">
              <MicIcon fontSize="small" sx={{ color: 'text.secondary' }} />
              <Box
                sx={{
                  flex: 1,
                  height: 6,
                  borderRadius: 3,
                  bgcolor: 'rgba(255,255,255,0.05)',
                  overflow: 'hidden',
                }}
              >
                <Box
                  sx={{
                    width: `${meterPct}%`,
                    height: '100%',
                    background: tokens.gradients.primary,
                    transition: 'width 80ms linear',
                  }}
                />
              </Box>
            </Stack>
          </Box>
        )}

        <Stack
          direction={{ xs: 'column', sm: 'row' }}
          spacing={1.5}
          sx={{ width: '100%', maxWidth: 360 }}
        >
          <Button
            variant="outlined"
            fullWidth
            onClick={onCancel}
            sx={{ borderColor: 'rgba(255,255,255,0.12)' }}
          >
            Cancel
          </Button>
          <Button
            variant="contained"
            fullWidth
            onClick={onJoin}
            sx={{
              background: tokens.gradients.primary,
              color: '#0a0b12',
              fontWeight: 600,
              boxShadow: tokens.shadows.glowPrimary,
              '&:hover': {
                background: tokens.gradients.primaryHover,
                boxShadow: tokens.shadows.glowPrimaryHover,
              },
            }}
          >
            Join meeting
          </Button>
        </Stack>

        <Typography
          variant="caption"
          color="text.secondary"
          sx={{ textAlign: 'center', maxWidth: 380 }}
        >
          Your camera and microphone start off. Toggle them on above to test before joining.
        </Typography>
      </Stack>
    </Box>
  )
}
