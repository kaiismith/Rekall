import { IconButton, Tooltip } from '@mui/material'
import ClosedCaptionIcon from '@mui/icons-material/ClosedCaption'
import ClosedCaptionDisabledIcon from '@mui/icons-material/ClosedCaptionDisabled'

interface CaptionsButtonProps {
  enabled: boolean
  onToggle: () => void
}

/**
 * Per-user toggle for the live-captions panel. Each participant controls
 * their own captions independently — there is no host gate or meeting-wide
 * flag. Turning it on opens your captions panel and starts transcribing
 * your own mic so other captioned participants can see what you say.
 */
export function CaptionsButton({ enabled, onToggle }: CaptionsButtonProps) {
  return (
    <Tooltip title={enabled ? 'Turn off live captions' : 'Turn on live captions'}>
      <IconButton
        onClick={onToggle}
        size="medium"
        aria-label="Toggle live captions"
        sx={{
          bgcolor: enabled ? 'primary.main' : 'action.selected',
          color: 'white',
          '&:hover': { bgcolor: enabled ? 'primary.dark' : 'action.hover' },
        }}
      >
        {enabled ? <ClosedCaptionIcon /> : <ClosedCaptionDisabledIcon />}
      </IconButton>
    </Tooltip>
  )
}
