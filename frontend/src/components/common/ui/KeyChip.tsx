import { Box, type SxProps, type Theme } from '@mui/material'
import type { ReactNode } from 'react'
import { tokens } from '@/theme'

interface KeyChipProps {
  children: ReactNode
  sx?: SxProps<Theme>
  className?: string
}

export function KeyChip({ children, sx, className }: KeyChipProps) {
  return (
    <Box
      component="kbd"
      className={className}
      sx={[
        {
          display: 'inline-flex',
          alignItems: 'center',
          padding: '2px 6px',
          fontFamily: tokens.fonts.mono,
          fontSize: '0.75rem',
          lineHeight: 1,
          border: '1px solid',
          borderColor: 'divider',
          borderRadius: `${tokens.radii.keyChip}px`,
          bgcolor: 'background.paper',
          color: 'text.primary',
        },
        ...(Array.isArray(sx) ? sx : [sx]),
      ]}
    >
      {children}
    </Box>
  )
}
