import { IconButton, Tooltip } from '@mui/material'
import VideocamIcon from '@mui/icons-material/Videocam'
import VideocamOffIcon from '@mui/icons-material/VideocamOff'

interface CameraButtonProps {
  isCameraOff: boolean
  onToggle: () => void
}

export function CameraButton({ isCameraOff, onToggle }: CameraButtonProps) {
  return (
    <Tooltip title={isCameraOff ? 'Turn camera on' : 'Turn camera off'}>
      <IconButton
        onClick={onToggle}
        size="medium"
        sx={{
          bgcolor: isCameraOff ? 'error.main' : 'action.selected',
          color: 'white',
          '&:hover': { bgcolor: isCameraOff ? 'error.dark' : 'action.hover' },
        }}
      >
        {isCameraOff ? <VideocamOffIcon /> : <VideocamIcon />}
      </IconButton>
    </Tooltip>
  )
}
