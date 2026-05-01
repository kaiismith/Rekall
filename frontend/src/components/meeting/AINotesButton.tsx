import { IconButton, Tooltip } from '@mui/material'
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome'
import AutoAwesomeOutlinedIcon from '@mui/icons-material/AutoAwesomeOutlined'

interface AINotesButtonProps {
  enabled: boolean
  onToggle: () => void
}

/**
 * Per-user toggle for the Kat live-notes panel. Independent of the captions
 * toggle: transcription runs unconditionally so Kat always has data, this
 * button only controls whether the Kat panel is visible in the right rail.
 *
 * Flipping it does NOT start or stop the underlying ASR pipeline; it only
 * shows / hides the rendered notes for this participant.
 */
export function AINotesButton({ enabled, onToggle }: AINotesButtonProps) {
  return (
    <Tooltip title={enabled ? 'Hide AI notes' : 'Show AI notes'}>
      <IconButton
        onClick={onToggle}
        size="medium"
        aria-label="Toggle AI notes"
        sx={{
          bgcolor: enabled ? 'primary.main' : 'action.selected',
          color: 'white',
          '&:hover': { bgcolor: enabled ? 'primary.dark' : 'action.hover' },
        }}
      >
        {enabled ? <AutoAwesomeIcon /> : <AutoAwesomeOutlinedIcon />}
      </IconButton>
    </Tooltip>
  )
}
