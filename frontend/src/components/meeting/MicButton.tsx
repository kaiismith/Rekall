import { Box, IconButton, Tooltip } from '@mui/material'
import MicIcon from '@mui/icons-material/Mic'
import MicOffIcon from '@mui/icons-material/MicOff'

interface MicButtonProps {
  isMuted: boolean
  audioLevel: number  // 0–1, drives the pulsing ring
  onToggle: () => void
}

export function MicButton({ isMuted, audioLevel, onToggle }: MicButtonProps) {
  // Ring animates with audio level: scale 1.0→1.4, opacity 0.2→1.0
  const ringScale = 1 + audioLevel * 0.4
  const ringOpacity = 0.2 + audioLevel * 0.8
  const showRing = !isMuted && audioLevel > 0.05

  return (
    <Tooltip title={isMuted ? 'Unmute' : 'Mute'}>
      <Box sx={{ position: 'relative', display: 'inline-flex' }}>
        {showRing && (
          <Box
            sx={{
              position: 'absolute',
              inset: -4,
              borderRadius: '50%',
              border: '2px solid',
              borderColor: 'primary.main',
              transform: `scale(${ringScale})`,
              opacity: ringOpacity,
              transition: 'transform 0.06s ease-out, opacity 0.06s ease-out',
              pointerEvents: 'none',
            }}
          />
        )}
        <IconButton
          onClick={onToggle}
          size="medium"
          sx={{
            bgcolor: isMuted ? 'error.main' : 'action.selected',
            color: 'white',
            '&:hover': { bgcolor: isMuted ? 'error.dark' : 'action.hover' },
          }}
        >
          {isMuted ? <MicOffIcon /> : <MicIcon />}
        </IconButton>
      </Box>
    </Tooltip>
  )
}
