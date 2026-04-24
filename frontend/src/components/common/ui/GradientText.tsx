import { Box, type SxProps, type Theme } from '@mui/material'
import type { ElementType, ReactNode } from 'react'
import { tokens } from '@/theme'

interface GradientTextProps {
  children: ReactNode
  component?: ElementType
  sx?: SxProps<Theme>
  className?: string
}

/**
 * Inline text rendered with the primary violet→blue gradient as its fill.
 * Use sparingly — one accent word per screen keeps the effect premium
 * rather than gaudy. Pairs well with the neutral "Welcome back, " pattern.
 */
export function GradientText({
  children,
  component = 'span',
  sx,
  className,
}: GradientTextProps) {
  return (
    <Box
      component={component}
      className={className}
      sx={[
        {
          display: 'inline',
          backgroundImage: tokens.gradients.primary,
          WebkitBackgroundClip: 'text',
          backgroundClip: 'text',
          WebkitTextFillColor: 'transparent',
          color: 'transparent',
        },
        ...(Array.isArray(sx) ? sx : [sx]),
      ]}
    >
      {children}
    </Box>
  )
}
