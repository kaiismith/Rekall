import { Box, Typography, type SxProps, type Theme } from '@mui/material'

interface OrDividerProps {
  sx?: SxProps<Theme>
  className?: string
}

export function OrDivider({ sx, className }: OrDividerProps) {
  return (
    <Box
      role="separator"
      aria-orientation="horizontal"
      className={className}
      sx={[
        { display: 'flex', alignItems: 'center', my: 3 },
        ...(Array.isArray(sx) ? sx : [sx]),
      ]}
    >
      <Box sx={{ flex: 1, height: '1px', bgcolor: 'divider' }} />
      <Typography
        variant="caption"
        sx={{ mx: 2, color: 'text.secondary', letterSpacing: '0.08em' }}
      >
        OR
      </Typography>
      <Box sx={{ flex: 1, height: '1px', bgcolor: 'divider' }} />
    </Box>
  )
}
