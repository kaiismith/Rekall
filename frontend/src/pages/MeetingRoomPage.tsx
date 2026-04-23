import { useCallback, useEffect, useRef } from 'react'
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
import BackHandIcon from '@mui/icons-material/BackHand'
import { useMeeting } from '@/hooks/useMeeting'
import { useAuthStore } from '@/store/authStore'
import { MicButton } from '@/components/meeting/MicButton'
import { CameraButton } from '@/components/meeting/CameraButton'
import { ShareButton } from '@/components/meeting/ShareButton'
import { HandButton } from '@/components/meeting/HandButton'
import { EmojiButton } from '@/components/meeting/EmojiButton'
import { BackgroundButton } from '@/components/meeting/BackgroundButton'
import { LaserButton } from '@/components/meeting/LaserButton'
import { ChatButton } from '@/components/meeting/ChatButton'
import { ChatPanel } from '@/components/meeting/chat/ChatPanel'
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

  const {
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
    handRaisedCount,
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
  } = useMeeting({
    code: code ?? '',
    onEnd: () => navigate('/meetings'),
  })

  // Attach local stream to video element.
  useEffect(() => {
    if (localVideoRef.current && localStream) {
      localVideoRef.current.srcObject = localStream
    }
  }, [localStream])

  // Laser pointer mouse handler on the video grid.
  const handleGridMouseMove = useCallback((e: React.MouseEvent<HTMLDivElement>) => {
    if (!isLaserActive || !videoGridRef.current) return
    const rect = videoGridRef.current.getBoundingClientRect()
    const x = (e.clientX - rect.left) / rect.width
    const y = (e.clientY - rect.top) / rect.height
    sendLaserMove(
      Math.max(0, Math.min(1, x)),
      Math.max(0, Math.min(1, y)),
    )
  }, [isLaserActive, sendLaserMove])

  // ── Connecting / waiting / denied / ended ─────────────────────────────────

  if (roomState === 'connecting') {
    return (
      <Box display="flex" flexDirection="column" alignItems="center" justifyContent="center" height="100vh" gap={2}>
        <CircularProgress />
        <Typography color="text.secondary">Connecting…</Typography>
      </Box>
    )
  }

  if (roomState === 'waiting_room') {
    return (
      <Box display="flex" flexDirection="column" alignItems="center" justifyContent="center" height="100vh" gap={3}>
        <Typography variant="h5">Waiting for admission</Typography>
        <Typography color="text.secondary">A participant will admit you shortly.</Typography>
        <CircularProgress size={28} />
      </Box>
    )
  }

  if (roomState === 'denied') {
    return (
      <Box display="flex" flexDirection="column" alignItems="center" justifyContent="center" height="100vh" gap={2}>
        <Typography variant="h5" color="error">Access Denied</Typography>
        <Typography color="text.secondary">Your request to join was declined.</Typography>
        <Button variant="outlined" onClick={() => navigate('/meetings')}>Back to Meetings</Button>
      </Box>
    )
  }

  if (roomState === 'ended' || roomState === 'error') {
    return (
      <Box display="flex" flexDirection="column" alignItems="center" justifyContent="center" height="100vh" gap={2}>
        <Typography variant="h5">Meeting Ended</Typography>
        <Button variant="outlined" onClick={() => navigate('/meetings')}>Back to Meetings</Button>
      </Box>
    )
  }

  // ── In-meeting UI ─────────────────────────────────────────────────────────

  const isHost = meeting?.host_id === user?.id
  const peerIds = Object.keys(peers)

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100vh', bgcolor: 'background.default' }}>

      {/* ── Top bar ─────────────────────────────────────────────────────────── */}
      <Box sx={{ px: 3, py: 1.5, borderBottom: '1px solid', borderColor: 'divider', display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexShrink: 0 }}>
        <Stack direction="row" spacing={1} alignItems="center">
          <Typography variant="h6" fontWeight={600}>{meeting?.title || 'Meeting'}</Typography>
          <Chip label={meeting?.code} size="small" sx={{ fontFamily: 'monospace' }} />
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
            <Button variant="contained" color="error" startIcon={<CallEndIcon />} onClick={endMeeting} size="small">
              End Meeting
            </Button>
          ) : (
            <Button variant="outlined" color="error" startIcon={<CallEndIcon />} onClick={leave} size="small">
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
          onMouseMove={handleGridMouseMove}
          sx={{
            flex: 1,
            position: 'relative',
            p: 2,
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))',
            gap: 2,
            alignContent: 'start',
            cursor: isLaserActive ? 'crosshair' : 'default',
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
            isHandRaised={false}
            reactions={reactionQueue.filter((r) => r.userId === user?.id)}
            isLocal
          >
            {!isCameraOff && (
              <video
                ref={localVideoRef}
                autoPlay
                muted
                playsInline
                style={{ width: '100%', height: '100%', objectFit: 'cover', transform: 'scaleX(-1)' }}
              />
            )}
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

          {/* Laser pointer overlay — spans entire grid */}
          {laserState && (
            <Box
              sx={{
                position: 'absolute',
                inset: 0,
                pointerEvents: 'none',
                zIndex: 10,
              }}
            >
              <Box
                sx={{
                  position: 'absolute',
                  left: `calc(${laserState.x * 100}% - 6px)`,
                  top: `calc(${laserState.y * 100}% - 6px)`,
                  width: 12,
                  height: 12,
                  borderRadius: '50%',
                  bgcolor: '#ef4444',
                  boxShadow: '0 0 8px 3px rgba(239,68,68,0.55)',
                  transition: 'left 0.02s linear, top 0.02s linear',
                }}
              />
            </Box>
          )}
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
          <Box sx={{ width: 280, borderLeft: '1px solid', borderColor: 'divider', p: 2, overflowY: 'auto', flexShrink: 0 }}>
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
                    <Button size="small" variant="contained" color="success" startIcon={<CheckIcon />} fullWidth
                      onClick={() => respondToKnock(knock.knock_id, true)}>
                      Admit
                    </Button>
                    <Button size="small" variant="outlined" color="error" startIcon={<CloseIcon />} fullWidth
                      onClick={() => respondToKnock(knock.knock_id, false)}>
                      Deny
                    </Button>
                  </Stack>
                </Paper>
              ))}
            </Stack>
          </Box>
        )}
      </Box>

      {/* ── Control bar ─────────────────────────────────────────────────────── */}
      <Box sx={{
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
      }}>
        <MicButton isMuted={isMuted} audioLevel={audioLevel} onToggle={toggleMute} />
        <CameraButton isCameraOff={isCameraOff} onToggle={toggleCamera} />
        <ShareButton isScreenSharing={isScreenSharing} onShare={shareScreen} onStop={stopScreenShare} />
        <HandButton isHandRaised={isHandRaised} onToggle={toggleHand} />
        <EmojiButton onSend={sendEmojiReaction} />
        <BackgroundButton
          active={activeBackground}
          onSelect={setBackground}
          onUpload={uploadCustomBackground}
          customBgSrc={customBgSrc}
          disabled={!bgSupported}
        />
        <LaserButton isActive={isLaserActive} onToggle={toggleLaser} />
        <ChatButton
          unreadCount={unreadCount}
          isOpen={isChatPanelOpen}
          onToggle={isChatPanelOpen ? closeChatPanel : openChatPanel}
        />
      </Box>
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
  hostAction?: () => void   // mute-other button shown to host
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
  const borderWidth = isLocal
    ? `${2 + audioLevel * 2}px`
    : '2px'

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
      <Box sx={{ position: 'absolute', bottom: 8, left: 8, display: 'flex', alignItems: 'center', gap: 0.5 }}>
        {isMuted
          ? <MicOffIcon fontSize="small" sx={{ color: 'error.light' }} />
          : (isSpeaking && <MicIcon fontSize="small" color="primary" />)
        }
        <Typography variant="caption" sx={{ bgcolor: 'rgba(0,0,0,0.55)', px: 0.5, borderRadius: 0.5 }}>
          {label}
        </Typography>
      </Box>

      {/* Hand raise — top right */}
      {isHandRaised && (
        <Box sx={{ position: 'absolute', top: 8, right: 8 }}>
          <Typography fontSize="1.2rem" lineHeight={1}>✋</Typography>
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
