import { useMemo } from 'react'
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Paper,
  Stack,
  Typography,
} from '@mui/material'
import MicIcon from '@mui/icons-material/Mic'
import StopIcon from '@mui/icons-material/Stop'

import { useASR, type ASRSessionKind } from '@/hooks/useASR'

interface LiveCaptionsProps {
  /** Call UUID or meeting code, depending on `kind`. */
  callId: string
  /** Defaults to `call`. Pass `meeting` to use the meeting endpoint pair. */
  kind?: ASRSessionKind
}

/**
 * LiveCaptions renders the rolling final transcript plus the live partial
 * for the currently-active segment, with a single Start/Stop control. The
 * component degrades gracefully when the ASR backend is not configured —
 * the hook surfaces ASR_NOT_CONFIGURED via its `error` channel and we hide
 * the panel.
 */
export function LiveCaptions({ callId, kind = 'call' }: LiveCaptionsProps) {
  const { state, partial, finals, error, start, stop } = useASR(callId, kind)

  const finalText = useMemo(() => finals.map((f) => f.text).join(' '), [finals])

  if (error?.code === 'ASR_NOT_CONFIGURED') return null

  const isStreaming = state === 'streaming' || state === 'reconnecting'
  const isBusy      = state === 'requesting' || state === 'connecting'

  return (
    <Paper variant="outlined" sx={{ p: 2 }}>
      <Stack spacing={2}>
        <Stack direction="row" alignItems="center" spacing={1}>
          <Typography variant="subtitle1" sx={{ flexGrow: 1 }}>
            Live captions
          </Typography>
          {isStreaming ? (
            <Button
              size="small"
              color="error"
              variant="outlined"
              startIcon={<StopIcon />}
              onClick={() => { void stop() }}
            >
              Stop
            </Button>
          ) : (
            <Button
              size="small"
              color="primary"
              variant="contained"
              startIcon={isBusy ? <CircularProgress size={16} color="inherit" /> : <MicIcon />}
              disabled={isBusy}
              onClick={() => { void start() }}
            >
              {isBusy ? 'Starting…' : 'Start captions'}
            </Button>
          )}
        </Stack>

        {state === 'reconnecting' && (
          <Alert severity="warning" variant="outlined">
            Reconnecting to the captions service…
          </Alert>
        )}
        {error && error.code !== 'ASR_NOT_CONFIGURED' && (
          <Alert severity="error" variant="outlined">
            {error.message}
          </Alert>
        )}

        <Box sx={{ minHeight: 96, fontFamily: 'inherit', whiteSpace: 'pre-wrap' }}>
          <Typography component="span" variant="body1">
            {finalText}
          </Typography>
          {partial && (
            <Typography
              component="span"
              variant="body1"
              sx={{ ml: finalText ? 1 : 0, opacity: 0.55, fontStyle: 'italic' }}
            >
              {partial}
            </Typography>
          )}
          {!finalText && !partial && state === 'streaming' && (
            <Typography variant="body2" color="text.secondary">
              Listening…
            </Typography>
          )}
          {!finalText && !partial && state !== 'streaming' && (
            <Typography variant="body2" color="text.secondary">
              Press “Start captions” to begin transcribing.
            </Typography>
          )}
        </Box>
      </Stack>
    </Paper>
  )
}
