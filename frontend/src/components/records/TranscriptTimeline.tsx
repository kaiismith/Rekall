import { useMemo } from 'react'
import {
  Avatar,
  Box,
  Button,
  CircularProgress,
  Skeleton,
  Stack,
  Tooltip,
  Typography,
} from '@mui/material'
import { useRecordTranscript } from '@/hooks/useRecordTranscript'
import { avatarColour, getInitials } from '@/utils'
import type { Meeting, SpeakerInfo } from '@/types/meeting'
import { groupBySpeaker, formatTimestamp } from './transcriptGrouping'

const UNKNOWN_SPEAKER: SpeakerInfo = {
  user_id: 'unknown',
  full_name: 'Unknown speaker',
  initials: '?',
}

const LOW_CONFIDENCE_THRESHOLD = 0.4

interface Props {
  code: string
  meeting: Meeting
}

export function TranscriptTimeline({ code, meeting }: Props) {
  const { segments, isLoading, isError, isFetchingNextPage, hasNextPage, loadMore } =
    useRecordTranscript(code)

  const speakerMap = useMemo(() => {
    const m = new Map<string, SpeakerInfo>()
    for (const s of meeting.speakers ?? []) {
      m.set(s.user_id, s)
    }
    return m
  }, [meeting.speakers])

  const blocks = useMemo(() => groupBySpeaker(segments), [segments])

  if (isLoading) {
    return (
      <Stack spacing={2} aria-label="Loading transcript">
        {[0, 1, 2].map((i) => (
          <Stack key={i} direction="row" spacing={2} alignItems="flex-start">
            <Skeleton variant="circular" width={32} height={32} />
            <Box flex={1}>
              <Skeleton variant="text" width="40%" />
              <Skeleton variant="text" width="90%" />
              <Skeleton variant="text" width="70%" />
            </Box>
          </Stack>
        ))}
      </Stack>
    )
  }

  if (isError) {
    return (
      <Box sx={{ p: 3, textAlign: 'center' }}>
        <Typography color="text.secondary">
          Couldn’t load this record’s transcript. Try refreshing.
        </Typography>
      </Box>
    )
  }

  if (segments.length === 0) {
    return (
      <Box sx={{ p: 3, textAlign: 'center' }}>
        <Typography color="text.secondary">No transcript available for this record</Typography>
      </Box>
    )
  }

  return (
    <Stack spacing={1.5}>
      {blocks.map((block) => {
        const speaker = speakerMap.get(block.speakerId) ?? UNKNOWN_SPEAKER
        const isUnknown = !speakerMap.has(block.speakerId)
        const initials = isUnknown ? '?' : speaker.initials || getInitials(speaker.full_name)
        const colour = isUnknown ? '#4b5563' : avatarColour(block.speakerId)

        return (
          <Box
            key={`${block.speakerId}-${block.startMs}-${block.segments[0]?.id}`}
            sx={{
              bgcolor: 'background.paper',
              borderRadius: 2,
              p: 2,
            }}
          >
            <Stack direction="row" spacing={1.5} alignItems="center" mb={1}>
              <Avatar sx={{ bgcolor: colour, width: 32, height: 32, fontSize: 13 }}>
                {initials}
              </Avatar>
              <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                {speaker.full_name}
              </Typography>
              <Typography variant="caption" color="text.secondary">
                {formatTimestamp(block.startMs)}
              </Typography>
            </Stack>

            <Stack spacing={0.75} pl={5.5}>
              {block.segments.map((seg) => {
                const lowConfidence =
                  seg.confidence != null && seg.confidence < LOW_CONFIDENCE_THRESHOLD
                const text = (
                  <Typography variant="body2" sx={{ color: lowConfidence ? '#9ca3af' : '#e5e7eb' }}>
                    {seg.text}
                  </Typography>
                )
                return (
                  <Box key={seg.id}>
                    {lowConfidence ? (
                      <Tooltip title="Low confidence" placement="left">
                        {text}
                      </Tooltip>
                    ) : (
                      text
                    )}
                  </Box>
                )
              })}
            </Stack>
          </Box>
        )
      })}

      {hasNextPage && (
        <Box sx={{ display: 'flex', justifyContent: 'center', pt: 1 }}>
          <Button
            variant="outlined"
            onClick={loadMore}
            disabled={isFetchingNextPage}
            startIcon={isFetchingNextPage ? <CircularProgress size={14} /> : undefined}
          >
            {isFetchingNextPage ? 'Loading…' : 'Load more'}
          </Button>
        </Box>
      )}
    </Stack>
  )
}
