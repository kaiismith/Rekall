import { Box, Chip, Stack, Typography } from '@mui/material'
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome'
import HelpOutlineIcon from '@mui/icons-material/HelpOutline'

import type { KatHealthResponse, KatNoteDTO, KatStatus } from '@/types/kat'

interface KatPanelProps {
  status: KatStatus
  latestNote: KatNoteDTO | null
  health: KatHealthResponse | null
}

/** Kat live notes panel.
 *
 *  Renders one of five visual states:
 *    - `idle`        — bootstrap probe in flight; show a subtle skeleton
 *    - `warming_up`  — Foundry configured, no notes yet ("Kat is listening…")
 *    - `live`        — render the latest summary + bullets + open questions
 *    - `offline`     — Foundry not configured; greyed card with operator hint
 *    - `error`       — probe failed; show a small notice
 *
 *  The "Notes are not saved — they live only during this meeting" hint is
 *  visible in `live` and `warming_up` so the ephemerality is observable to
 *  the user without reading any docs.
 */
export function KatPanel({ status, latestNote, health }: KatPanelProps) {
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <Stack
        direction="row"
        alignItems="center"
        spacing={1}
        sx={{ pb: 1.5, borderBottom: '1px solid', borderColor: 'divider', flexShrink: 0 }}
      >
        <AutoAwesomeIcon fontSize="small" sx={{ color: 'primary.main' }} />
        <Typography variant="subtitle2" fontWeight={600} sx={{ flexGrow: 1 }}>
          Kat — live notes
        </Typography>
        {status === 'live' && <Chip label="Live" size="small" color="success" variant="outlined" />}
        {status === 'warming_up' && <Chip label="Warming up" size="small" variant="outlined" />}
        {status === 'offline' && <Chip label="Offline" size="small" variant="outlined" />}
        {status === 'error' && (
          <Chip label="Error" size="small" color="warning" variant="outlined" />
        )}
      </Stack>

      <Box sx={{ flex: 1, overflowY: 'auto', mt: 2, pr: 1 }}>
        {renderBody({ status, latestNote })}
      </Box>

      {(status === 'live' || status === 'warming_up') && (
        <Stack
          spacing={0.25}
          sx={{
            pt: 1.5,
            mt: 1,
            borderTop: '1px solid',
            borderColor: 'divider',
            flexShrink: 0,
          }}
        >
          <Typography variant="caption" color="text.secondary">
            Notes are not saved — they live only during this meeting.
          </Typography>
          <Typography variant="caption" color="text.disabled">
            Powered by Kat
            {health?.deployment ? ` • model: ${health.deployment}` : ''}
          </Typography>
        </Stack>
      )}
    </Box>
  )
}

function renderBody({ status, latestNote }: Pick<KatPanelProps, 'status' | 'latestNote'>) {
  if (status === 'offline') {
    return (
      <Box>
        <Typography variant="body2" color="text.secondary">
          Kat is offline.
        </Typography>
        <Typography variant="caption" color="text.disabled">
          Ask your administrator to configure AI Foundry.
        </Typography>
      </Box>
    )
  }

  if (status === 'error') {
    return (
      <Typography variant="body2" color="warning.main">
        Couldn&apos;t reach Kat. The captions UX is unaffected.
      </Typography>
    )
  }

  if (status === 'idle' || (status === 'warming_up' && !latestNote)) {
    return (
      <Typography variant="body2" color="text.secondary">
        Kat is listening… first notes in a moment.
      </Typography>
    )
  }

  if (!latestNote) {
    return null
  }

  return (
    <Stack spacing={2} data-testid="kat-note">
      <Box>
        <Typography variant="caption" color="text.disabled" sx={{ display: 'block', mb: 0.5 }}>
          Updated {formatTimestamp(latestNote.window_ended_at)}
          {' · '}covers last {formatDuration(latestNote)}
        </Typography>
        <Typography variant="body1" data-testid="kat-summary">
          {latestNote.summary}
        </Typography>
      </Box>

      {latestNote.key_points.length > 0 && (
        <Box>
          <Typography variant="overline" color="text.secondary">
            Key points
          </Typography>
          <Box component="ul" sx={{ pl: 3, m: 0 }} data-testid="kat-key-points">
            {latestNote.key_points.map((kp, i) => (
              <Typography component="li" variant="body2" key={i}>
                {kp}
              </Typography>
            ))}
          </Box>
        </Box>
      )}

      {latestNote.open_questions.length > 0 && (
        <Box>
          <Typography variant="overline" color="text.secondary">
            Open questions
          </Typography>
          <Stack spacing={0.5} data-testid="kat-open-questions">
            {latestNote.open_questions.map((q, i) => (
              <Stack direction="row" spacing={1} key={i} alignItems="flex-start">
                <HelpOutlineIcon fontSize="small" color="info" />
                <Typography variant="body2" sx={{ fontStyle: 'italic' }}>
                  {q}
                </Typography>
              </Stack>
            ))}
          </Stack>
        </Box>
      )}
    </Stack>
  )
}

function formatTimestamp(iso: string): string {
  try {
    const d = new Date(iso)
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  } catch {
    return iso
  }
}

function formatDuration(note: KatNoteDTO): string {
  try {
    const start = new Date(note.window_started_at).getTime()
    const end = new Date(note.window_ended_at).getTime()
    const sec = Math.max(0, Math.round((end - start) / 1000))
    if (sec < 60) return `${sec}s`
    const min = Math.round(sec / 60)
    return `${min} min`
  } catch {
    return '—'
  }
}
