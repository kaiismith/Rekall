import { IconButton, Tooltip } from '@mui/material'
import ScreenShareIcon from '@mui/icons-material/ScreenShare'
import StopScreenShareIcon from '@mui/icons-material/StopScreenShare'

interface ShareButtonProps {
  isScreenSharing: boolean
  onShare: () => void
  onStop: () => void
}

export function ShareButton({ isScreenSharing, onShare, onStop }: ShareButtonProps) {
  return (
    <Tooltip title={isScreenSharing ? 'Stop sharing' : 'Share screen'}>
      <IconButton
        onClick={isScreenSharing ? onStop : onShare}
        size="medium"
        sx={{
          bgcolor: isScreenSharing ? 'primary.main' : 'action.selected',
          color: 'white',
          '&:hover': { bgcolor: isScreenSharing ? 'primary.dark' : 'action.hover' },
        }}
      >
        {isScreenSharing ? <StopScreenShareIcon /> : <ScreenShareIcon />}
      </IconButton>
    </Tooltip>
  )
}
