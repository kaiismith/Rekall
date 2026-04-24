import { Paper, type SxProps, type Theme } from '@mui/material'
import type { ReactNode } from 'react'
import { tokens } from '@/theme'

interface ActionCardProps {
  children: ReactNode
  /** Max horizontal width in pixels. Defaults to 520. */
  maxWidth?: number
  sx?: SxProps<Theme>
  className?: string
}

export function ActionCard({
  children,
  maxWidth = 520,
  sx,
  className,
}: ActionCardProps) {
  return (
    <Paper
      className={className}
      sx={[
        {
          borderRadius: `${tokens.radii.card}px`,
          boxShadow: tokens.shadows.elevatedCard,
          background: tokens.gradients.cardSurface,
          padding: { xs: 3, sm: 4.5 },
          border: '1px solid rgba(255,255,255,0.06)',
          backdropFilter: 'blur(8px)',
          maxWidth,
          width: '100%',
          mx: 'auto',
        },
        ...(Array.isArray(sx) ? sx : [sx]),
      ]}
    >
      {children}
    </Paper>
  )
}
