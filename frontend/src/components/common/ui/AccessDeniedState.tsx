import { Box, Typography } from '@mui/material'
import LockOutlinedIcon from '@mui/icons-material/LockOutlined'
import { useNavigate } from 'react-router-dom'
import { ROUTES } from '@/constants'
import { GradientButton } from './GradientButton'

/**
 * Rendered by scoped pages when the primary fetch returns 403/404.
 * Lives inside the authenticated Layout so the sidebar/breadcrumb context is
 * preserved — the user can climb back up the hierarchy without losing place.
 */
export function AccessDeniedState() {
  const navigate = useNavigate()
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 3,
        py: { xs: 8, sm: 12 },
        px: 3,
        textAlign: 'center',
      }}
    >
      <LockOutlinedIcon aria-hidden sx={{ fontSize: 64, color: 'text.disabled' }} />
      <Typography variant="h5" sx={{ fontWeight: 600, color: 'text.primary' }}>
        You don&apos;t have access to this space
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ maxWidth: 420 }}>
        This organization or department is private to its members. Ask an admin
        to add you, or return to your workspace.
      </Typography>
      <GradientButton
        fullWidth={false}
        onClick={() => navigate(ROUTES.DASHBOARD)}
        sx={{ minWidth: 200 }}
      >
        Back to workspace
      </GradientButton>
    </Box>
  )
}
