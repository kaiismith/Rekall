import { IconButton, Tooltip } from '@mui/material'
import AdjustIcon from '@mui/icons-material/Adjust'

interface LaserButtonProps {
  isActive: boolean
  onToggle: () => void
}

export function LaserButton({ isActive, onToggle }: LaserButtonProps) {
  return (
    <Tooltip title={isActive ? 'Stop laser pointer' : 'Laser pointer'}>
      <IconButton
        onClick={onToggle}
        size="medium"
        sx={{
          bgcolor: isActive ? 'error.main' : 'action.selected',
          color: 'white',
          '&:hover': { bgcolor: isActive ? 'error.dark' : 'action.hover' },
        }}
      >
        <AdjustIcon />
      </IconButton>
    </Tooltip>
  )
}
