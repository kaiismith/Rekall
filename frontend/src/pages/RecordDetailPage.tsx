import { useNavigate, useParams, Link as RouterLink } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { isAxiosError } from 'axios'
import {
  Alert,
  Box,
  Breadcrumbs,
  Button,
  CircularProgress,
  Link as MuiLink,
  Stack,
  Typography,
} from '@mui/material'
import ArrowBackIcon from '@mui/icons-material/ArrowBack'
import { meetingService } from '@/services/meetingService'
import { MeetingCard } from '@/components/meetings/MeetingCard'
import { TranscriptTimeline } from '@/components/records/TranscriptTimeline'
import { ROUTES } from '@/constants'
import type { Meeting } from '@/types/meeting'

function isNotFound(err: unknown): boolean {
  return isAxiosError(err) && err.response?.status === 404
}

function isForbidden(err: unknown): boolean {
  return isAxiosError(err) && err.response?.status === 403
}

function isInProgress(meeting: Meeting): boolean {
  return meeting.status === 'waiting' || meeting.status === 'active'
}

function BackLink() {
  return (
    <Button
      component={RouterLink}
      to={ROUTES.RECORDS}
      startIcon={<ArrowBackIcon />}
      size="small"
      sx={{ mb: 2 }}
    >
      Back to Records
    </Button>
  )
}

/**
 * Record detail — non-clickable MeetingCard header + paginated transcript
 * timeline. Auth gating mirrors the backend: host or past participant only.
 */
export function RecordDetailPage() {
  const { code = '' } = useParams<{ code: string }>()
  const navigate = useNavigate()

  const meetingQuery = useQuery({
    queryKey: ['meeting', code],
    queryFn: async () => {
      const resp = await meetingService.getByCode(code)
      return resp.data
    },
    enabled: Boolean(code),
    retry: (count, err) => {
      // Don't retry obvious user-facing failures.
      if (isNotFound(err) || isForbidden(err)) return false
      return count < 1
    },
  })

  if (meetingQuery.isLoading) {
    return (
      <Box sx={{ maxWidth: 960, mx: 'auto', display: 'flex', justifyContent: 'center', py: 8 }}>
        <CircularProgress />
      </Box>
    )
  }

  if (meetingQuery.isError) {
    const notFound = isNotFound(meetingQuery.error)
    const forbidden = isForbidden(meetingQuery.error)
    const title = notFound
      ? 'Record not found'
      : forbidden
        ? 'You don’t have access to this record'
        : 'Couldn’t load this record'
    const description = notFound
      ? 'This record doesn’t exist or has been deleted.'
      : forbidden
        ? 'Only the host or past participants can read this record.'
        : 'Something went wrong on our end. Please try again.'

    return (
      <Box sx={{ maxWidth: 960, mx: 'auto', py: 6 }}>
        <BackLink />
        <Stack spacing={1.5} alignItems="center" textAlign="center" py={6}>
          <Typography variant="h6">{title}</Typography>
          <Typography color="text.secondary">{description}</Typography>
        </Stack>
      </Box>
    )
  }

  const meeting = meetingQuery.data!

  const titleLabel = meeting.title || `Meeting ${meeting.code}`

  return (
    <Box sx={{ maxWidth: 960, mx: 'auto' }}>
      <BackLink />

      <Breadcrumbs sx={{ mb: 2 }} aria-label="breadcrumb">
        <MuiLink component={RouterLink} to={ROUTES.RECORDS} underline="hover" color="inherit">
          Records
        </MuiLink>
        <Typography color="text.primary">{titleLabel}</Typography>
      </Breadcrumbs>

      <Stack spacing={2}>
        <MeetingCard meeting={meeting} clickable={false} />

        {isInProgress(meeting) && (
          <Alert
            severity="info"
            action={
              <Button
                color="inherit"
                size="small"
                onClick={() => navigate(`/meeting/${meeting.code}`)}
              >
                Join live
              </Button>
            }
          >
            This record is still in progress. Refresh to see new segments.
          </Alert>
        )}

        <TranscriptTimeline code={code} meeting={meeting} />
      </Stack>
    </Box>
  )
}
