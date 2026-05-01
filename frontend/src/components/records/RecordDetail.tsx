import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { isAxiosError } from 'axios'
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Snackbar,
  Stack,
  Tab,
  Tabs,
  Typography,
} from '@mui/material'
import ShareIcon from '@mui/icons-material/Share'
import FileDownloadIcon from '@mui/icons-material/FileDownload'
import { meetingService } from '@/services/meetingService'
import { MeetingCard } from '@/components/meetings/MeetingCard'
import { TranscriptTimeline } from '@/components/records/TranscriptTimeline'
import type { Meeting } from '@/types/meeting'

type TabKey = 'summary' | 'transcript' | 'action_items'

function isNotFound(err: unknown): boolean {
  return isAxiosError(err) && err.response?.status === 404
}

function isForbidden(err: unknown): boolean {
  return isAxiosError(err) && err.response?.status === 403
}

function isInProgress(meeting: Meeting): boolean {
  return meeting.status === 'waiting' || meeting.status === 'active'
}

function ComingSoon({ feature }: { feature: string }) {
  return (
    <Box
      sx={{
        py: 8,
        px: 3,
        textAlign: 'center',
        bgcolor: 'background.paper',
        borderRadius: 2,
      }}
    >
      <Typography variant="h6" color="text.primary" gutterBottom>
        {feature} — coming soon
      </Typography>
      <Typography color="text.secondary">
        This area will be filled in by an upcoming spec.
      </Typography>
    </Box>
  )
}

interface Props {
  code: string
}

/**
 * Right-pane detail view used by the records two-pane layout. Loads the
 * meeting by code, exposes Share / Export, and renders Summary / Transcript /
 * Action Items tabs. Auth gating mirrors the backend.
 */
export function RecordDetail({ code }: Props) {
  const navigate = useNavigate()

  const [tab, setTab] = useState<TabKey>('transcript')
  const [shareToast, setShareToast] = useState<string | null>(null)

  const meetingQuery = useQuery({
    queryKey: ['meeting', code],
    queryFn: async () => {
      const resp = await meetingService.getByCode(code)
      return resp.data
    },
    enabled: Boolean(code),
    retry: (count, err) => {
      if (isNotFound(err) || isForbidden(err)) return false
      return count < 1
    },
  })

  const handleShare = async () => {
    const url = `${window.location.origin}/records/${code}`
    try {
      await navigator.clipboard.writeText(url)
      setShareToast('Record link copied to clipboard')
    } catch {
      setShareToast(`Copy this link: ${url}`)
    }
  }

  const handleExport = () => {
    setShareToast('Export — coming soon')
  }

  if (meetingQuery.isLoading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 8 }}>
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
      <Stack spacing={1.5} alignItems="center" textAlign="center" py={6}>
        <Typography variant="h6">{title}</Typography>
        <Typography color="text.secondary">{description}</Typography>
      </Stack>
    )
  }

  const meeting = meetingQuery.data!
  const titleLabel = meeting.title || `Meeting ${meeting.code}`

  return (
    <Box>
      <Stack
        direction="row"
        justifyContent="space-between"
        alignItems="flex-start"
        spacing={2}
        sx={{ mb: 2 }}
      >
        <Typography variant="h5" sx={{ fontWeight: 700, minWidth: 0 }} noWrap>
          {titleLabel}
        </Typography>
        <Stack direction="row" spacing={1} flexShrink={0}>
          <Button
            variant="outlined"
            size="small"
            startIcon={<ShareIcon />}
            onClick={() => void handleShare()}
          >
            Share
          </Button>
          <Button
            variant="contained"
            size="small"
            startIcon={<FileDownloadIcon />}
            onClick={handleExport}
          >
            Export
          </Button>
        </Stack>
      </Stack>

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

        <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
          <Tabs value={tab} onChange={(_, v: TabKey) => setTab(v)} aria-label="Record content tabs">
            <Tab value="summary" label="Summary" />
            <Tab value="transcript" label="Transcript" />
            <Tab value="action_items" label="Action Items" />
          </Tabs>
        </Box>

        {tab === 'summary' && <ComingSoon feature="Summary" />}
        {tab === 'transcript' && <TranscriptTimeline code={code} meeting={meeting} />}
        {tab === 'action_items' && <ComingSoon feature="Action items" />}
      </Stack>

      <Snackbar
        open={Boolean(shareToast)}
        autoHideDuration={3500}
        onClose={() => setShareToast(null)}
        message={shareToast ?? ''}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      />
    </Box>
  )
}

/**
 * Right-pane empty state — shown when no record is selected. Mirrors the
 * "Intelligence Ready" hero from the design.
 */
export function RecordsEmptyState() {
  return (
    <Stack spacing={3} alignItems="center" textAlign="center" sx={{ py: 8, px: 3 }}>
      <Box
        sx={{
          width: 96,
          height: 96,
          borderRadius: '20px',
          bgcolor: 'rgba(255,255,255,0.04)',
          border: '1px solid rgba(255,255,255,0.06)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          color: 'text.secondary',
          fontSize: '3rem',
        }}
      >
        💬
      </Box>
      <Typography variant="h4" sx={{ fontWeight: 700 }}>
        Intelligence Ready
      </Typography>
      <Typography color="text.secondary" sx={{ maxWidth: 480 }}>
        Select a recording from the list to view its transcript, automated executive summary, and
        sentiment analytics.
      </Typography>
    </Stack>
  )
}
