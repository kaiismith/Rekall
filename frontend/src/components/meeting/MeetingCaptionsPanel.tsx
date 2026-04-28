import { useEffect, useMemo, useRef } from 'react'
import {
  Avatar,
  Box,
  Chip,
  IconButton,
  Stack,
  Tooltip,
  Typography,
} from '@mui/material'
import ContentCopyIcon from '@mui/icons-material/ContentCopy'
import ClosedCaptionIcon from '@mui/icons-material/ClosedCaption'

import type { CaptionEntry, ParticipantDirectoryEntry } from '@/types/meeting'

interface MeetingCaptionsPanelProps {
  captions: CaptionEntry[]
  directory: Record<string, ParticipantDirectoryEntry>
  localUserId: string | null
  isStreamingLocal: boolean   // is this user actively running ASR?
  isDisabled: boolean          // host has turned transcription off
  onClose?: () => void
}

/**
 * MeetingCaptionsPanel renders the merged transcript feed for a meeting.
 *
 * Each entry shows the speaker's display name + a wall-clock timestamp.
 * Partials render in italic to signal they may still be revised; finals
 * render in normal weight. Auto-scrolls to the bottom when new content
 * arrives unless the user has scrolled up to read history.
 */
export function MeetingCaptionsPanel({
  captions,
  directory,
  localUserId,
  isStreamingLocal,
  isDisabled,
}: MeetingCaptionsPanelProps) {
  const scrollRef = useRef<HTMLDivElement>(null)
  const stickToBottomRef = useRef(true)

  // Track whether the user is at (or very near) the bottom. If they scroll
  // up to read history, stop auto-scrolling so we don't yank them back.
  useEffect(() => {
    const el = scrollRef.current
    if (!el) return
    const onScroll = () => {
      const distanceFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight
      stickToBottomRef.current = distanceFromBottom < 32
    }
    el.addEventListener('scroll', onScroll, { passive: true })
    return () => el.removeEventListener('scroll', onScroll)
  }, [])

  // Auto-scroll to bottom whenever the captions list grows AND the user
  // hasn't manually scrolled up.
  useEffect(() => {
    if (!stickToBottomRef.current) return
    const el = scrollRef.current
    if (el) el.scrollTop = el.scrollHeight
  }, [captions])

  const transcriptText = useMemo(() => (
    captions
      .filter((c) => c.kind === 'final')
      .map((c) => {
        const name = directory[c.userId]?.full_name ?? shortId(c.userId)
        return `[${formatTime(c.timestamp)}] ${name}: ${c.text}`
      })
      .join('\n')
  ), [captions, directory])

  const handleCopy = () => {
    if (!transcriptText) return
    void navigator.clipboard?.writeText(transcriptText).catch(() => { /* ignore */ })
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* ── Header ──────────────────────────────────────────────────────── */}
      <Stack
        direction="row"
        alignItems="center"
        spacing={1}
        sx={{ pb: 1.5, borderBottom: '1px solid', borderColor: 'divider', flexShrink: 0 }}
      >
        <ClosedCaptionIcon fontSize="small" sx={{ color: 'primary.main' }} />
        <Typography variant="subtitle2" fontWeight={600} sx={{ flexGrow: 1 }}>
          Live captions
        </Typography>
        {isStreamingLocal && !isDisabled && (
          <Chip label="Streaming" size="small" color="success" variant="outlined" />
        )}
        {isDisabled && (
          <Chip label="Off" size="small" variant="outlined" />
        )}
        <Tooltip title="Copy transcript">
          <span>
            <IconButton
              size="small"
              onClick={handleCopy}
              disabled={!transcriptText}
              aria-label="Copy transcript"
            >
              <ContentCopyIcon fontSize="small" />
            </IconButton>
          </span>
        </Tooltip>
      </Stack>

      {/* ── Feed ────────────────────────────────────────────────────────── */}
      <Box
        ref={scrollRef}
        sx={{
          flex: 1,
          overflowY: 'auto',
          mt: 1,
          pr: 1,
        }}
      >
        {captions.length === 0 ? (
          <Typography variant="body2" color="text.secondary" sx={{ mt: 2 }}>
            {isDisabled
              ? 'Transcription is off.'
              : 'Captions will appear here when participants speak.'}
          </Typography>
        ) : (
          <Stack spacing={1.25}>
            {captions.map((c) => {
              const entry = directory[c.userId]
              const name  = entry?.full_name ?? shortId(c.userId)
              const initials = entry?.initials ?? initialsFrom(name)
              const isMine = c.userId === localUserId
              return (
                <Stack key={c.key} direction="row" spacing={1} alignItems="flex-start">
                  <Avatar
                    sx={{
                      width: 24,
                      height: 24,
                      fontSize: '0.7rem',
                      bgcolor: isMine ? 'primary.dark' : 'grey.700',
                      flexShrink: 0,
                      mt: 0.25,
                    }}
                  >
                    {initials}
                  </Avatar>
                  <Box sx={{ minWidth: 0, flex: 1 }}>
                    <Stack direction="row" spacing={1} alignItems="baseline">
                      <Typography
                        variant="caption"
                        fontWeight={600}
                        sx={{ color: isMine ? 'primary.light' : 'text.primary' }}
                      >
                        {isMine ? 'You' : name}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        {formatTime(c.timestamp)}
                      </Typography>
                    </Stack>
                    <Typography
                      variant="body2"
                      sx={{
                        whiteSpace: 'pre-wrap',
                        wordBreak: 'break-word',
                        fontStyle: c.kind === 'partial' ? 'italic' : 'normal',
                        opacity:   c.kind === 'partial' ? 0.65       : 1,
                      }}
                    >
                      {c.text}
                    </Typography>
                  </Box>
                </Stack>
              )
            })}
          </Stack>
        )}
      </Box>
    </Box>
  )
}

function formatTime(ts: number): string {
  const d = new Date(ts)
  const hh = String(d.getHours()).padStart(2, '0')
  const mm = String(d.getMinutes()).padStart(2, '0')
  const ss = String(d.getSeconds()).padStart(2, '0')
  return `${hh}:${mm}:${ss}`
}

function shortId(uid: string): string {
  return uid.length > 8 ? uid.slice(0, 8) + '…' : uid
}

function initialsFrom(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase()
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
}
