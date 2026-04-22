import { IconButton, Tooltip } from '@mui/material'
import BackHandIcon from '@mui/icons-material/BackHand'
import BackHandOutlinedIcon from '@mui/icons-material/BackHandOutlined'

interface HandButtonProps {
  isHandRaised: boolean
  onToggle: () => void
}

export function HandButton({ isHandRaised, onToggle }: HandButtonProps) {
  return (
    <Tooltip title={isHandRaised ? 'Lower hand' : 'Raise hand'}>
      <IconButton
        onClick={onToggle}
        size="medium"
        sx={{
          bgcolor: isHandRaised ? 'warning.main' : 'action.selected',
          color: 'white',
          '&:hover': { bgcolor: isHandRaised ? 'warning.dark' : 'action.hover' },
        }}
      >
        {isHandRaised ? <BackHandIcon /> : <BackHandOutlinedIcon />}
      </IconButton>
    </Tooltip>
  )
}
