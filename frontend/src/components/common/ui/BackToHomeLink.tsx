import { Button, useMediaQuery, useTheme } from '@mui/material'
import ArrowBackIcon from '@mui/icons-material/ArrowBack'
import { useNavigate } from 'react-router-dom'
import { ROUTES } from '@/constants'

interface BackToHomeLinkProps {
  to?: string
  label?: string
}

export function BackToHomeLink({
  to = ROUTES.DASHBOARD,
  label = 'Back to home',
}: BackToHomeLinkProps) {
  const navigate = useNavigate()
  const theme = useTheme()
  const hideLabel = useMediaQuery(theme.breakpoints.down('sm'))

  return (
    <Button
      variant="outlined"
      size="small"
      startIcon={<ArrowBackIcon />}
      onClick={() => navigate(to)}
      sx={{
        borderColor: 'rgba(255,255,255,0.08)',
        bgcolor: 'rgba(255,255,255,0.02)',
        color: 'text.primary',
        textTransform: 'none',
        '&:hover': {
          borderColor: 'rgba(255,255,255,0.15)',
          bgcolor: 'rgba(255,255,255,0.05)',
        },
        '& .MuiButton-startIcon': { mr: hideLabel ? 0 : 1 },
        minWidth: hideLabel ? 36 : undefined,
        px: hideLabel ? 1 : 2,
      }}
    >
      {!hideLabel && label}
    </Button>
  )
}
