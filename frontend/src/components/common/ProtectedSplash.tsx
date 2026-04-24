import { Box, CircularProgress } from '@mui/material'

export function ProtectedSplash() {
  return (
    <Box
      data-testid="protected-splash"
      sx={{
        position: 'fixed',
        inset: 0,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        bgcolor: 'background.default',
        zIndex: (theme) => theme.zIndex.modal,
      }}
    >
      <CircularProgress size={28} thickness={4} />
    </Box>
  )
}
