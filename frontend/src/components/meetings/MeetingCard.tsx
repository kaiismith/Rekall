import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Box, Stack, Typography } from '@mui/material'
import type { Meeting, ParticipantPreview } from '@/types/meeting'
import { formatMeetingDuration, computeElapsed, stringToColor } from '@/utils'

// ─── Audio bar decorative graphic ────────────────────────────────────────────

function AudioBarGraphic() {
  const bars = [4, 8, 12, 6, 10, 7, 14, 5]
  return (
    <Box sx={{ display: 'flex', alignItems: 'flex-end', gap: '2px', height: 16, opacity: 0.35 }}>
      {bars.map((h, i) => (
        <Box
          key={i}
          sx={{ width: 3, height: h, bgcolor: 'text.secondary', borderRadius: '1px' }}
        />
      ))}
    </Box>
  )
}

// ─── Participant avatar stack ─────────────────────────────────────────────────

function AvatarStack({ previews }: { previews: ParticipantPreview[] }) {
  const visible = previews.slice(0, 3)
  const overflow = previews.length - visible.length
  return (
    <Stack direction="row" sx={{ position: 'relative' }}>
      {visible.map((p, i) => (
        <Box
          key={p.user_id}
          sx={{
            width: 28,
            height: 28,
            borderRadius: '50%',
            bgcolor: stringToColor(p.user_id),
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: 11,
            fontWeight: 700,
            color: '#fff',
            border: '2px solid',
            borderColor: 'background.paper',
            ml: i === 0 ? 0 : '-6px',
            zIndex: visible.length - i,
          }}
        >
          {p.initials}
        </Box>
      ))}
      {overflow > 0 && (
        <Box
          sx={{
            width: 28,
            height: 28,
            borderRadius: '50%',
            bgcolor: 'action.selected',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: 10,
            fontWeight: 600,
            color: 'text.secondary',
            border: '2px solid',
            borderColor: 'background.paper',
            ml: '-6px',
          }}
        >
          +{overflow}
        </Box>
      )}
    </Stack>
  )
}

// ─── Duration display ─────────────────────────────────────────────────────────

function Duration({ meeting }: { meeting: Meeting }) {
  const isLive = meeting.status === 'waiting' || meeting.status === 'active'

  const [elapsed, setElapsed] = useState(() =>
    isLive && meeting.started_at ? computeElapsed(meeting.started_at) : 0,
  )

  useEffect(() => {
    if (!isLive || !meeting.started_at) return
    const id = setInterval(() => setElapsed(computeElapsed(meeting.started_at!)), 1000)
    return () => clearInterval(id)
  }, [isLive, meeting.started_at])

  if (!meeting.started_at) return <span>—</span>
  if (isLive) return <span>{formatMeetingDuration(elapsed)}</span>
  if (meeting.duration_seconds != null) return <span>{formatMeetingDuration(meeting.duration_seconds)}</span>
  return <span>—</span>
}

// ─── MeetingCard ──────────────────────────────────────────────────────────────

interface Props {
  meeting: Meeting
}

export function MeetingCard({ meeting }: Props) {
  const navigate = useNavigate()
  const isLive = meeting.status === 'waiting' || meeting.status === 'active'
  const previews = meeting.participant_previews ?? []

  const formattedDate = new Intl.DateTimeFormat('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  })
    .format(new Date(meeting.created_at))
    .toUpperCase()

  return (
    <Box
      onClick={() => navigate(`/meeting/${meeting.code}`)}
      onKeyDown={(e) => e.key === 'Enter' && navigate(`/meeting/${meeting.code}`)}
      tabIndex={0}
      role="button"
      sx={{
        position: 'relative',
        bgcolor: 'background.paper',
        borderRadius: 2,
        p: 2.5,
        cursor: 'pointer',
        outline: 'none',
        '&:hover': { bgcolor: 'action.hover' },
        '&:focus-visible': { outline: '2px solid', outlineColor: 'primary.main' },
      }}
    >
      {/* Status dot */}
      {isLive && (
        <Box
          sx={{
            position: 'absolute',
            top: 16,
            left: 16,
            width: 8,
            height: 8,
            borderRadius: '50%',
            bgcolor: '#22c55e',
          }}
        />
      )}

      {/* Duration — top right */}
      <Typography
        variant="caption"
        sx={{
          position: 'absolute',
          top: 14,
          right: 16,
          fontWeight: 600,
          color: 'text.secondary',
          bgcolor: 'action.selected',
          px: 1,
          py: 0.25,
          borderRadius: 1,
          fontSize: 12,
        }}
      >
        <Duration meeting={meeting} />
      </Typography>

      {/* Title */}
      <Typography
        fontWeight={700}
        sx={{ mt: isLive ? 0.5 : 0, mb: 1, pr: 8, fontSize: 15 }}
      >
        {meeting.title || `Meeting ${meeting.code}`}
      </Typography>

      {/* Type badge + date */}
      <Stack direction="row" spacing={1.5} alignItems="center" mb={1.5}>
        <Box
          sx={{
            px: 1,
            py: 0.25,
            borderRadius: 1,
            bgcolor: '#2a2a3a',
            color: '#9ca3af',
            fontSize: 11,
            fontWeight: 700,
            letterSpacing: '0.05em',
          }}
        >
          {meeting.type.toUpperCase()}
        </Box>
        <Typography variant="caption" color="text.secondary" fontSize={12}>
          {formattedDate}
        </Typography>
      </Stack>

      {/* Avatar stack + audio bar */}
      <Stack direction="row" justifyContent="space-between" alignItems="center">
        <AvatarStack previews={previews} />
        <AudioBarGraphic />
      </Stack>
    </Box>
  )
}
