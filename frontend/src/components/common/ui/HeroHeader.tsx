import { Box, Typography, type SxProps, type Theme } from '@mui/material'
import type { ElementType } from 'react'
import { tokens } from '@/theme'

interface HeroHeaderProps {
  title: string
  subtitle?: string
  align?: 'center' | 'left'
  /** Defaults to 'h1' so the page exposes a single top-level landmark. */
  component?: ElementType
  sx?: SxProps<Theme>
  className?: string
}

export function HeroHeader({
  title,
  subtitle,
  align = 'center',
  component = 'h1',
  sx,
  className,
}: HeroHeaderProps) {
  return (
    <Box
      className={className}
      sx={[
        {
          position: 'relative',
          textAlign: align,
          py: 4,
          width: '100%',
        },
        ...(Array.isArray(sx) ? sx : [sx]),
      ]}
    >
      <Box
        aria-hidden
        sx={{
          position: 'absolute',
          inset: 0,
          background: tokens.gradients.heroBackground,
          pointerEvents: 'none',
          zIndex: 0,
        }}
      />
      <Box sx={{ position: 'relative', zIndex: 1 }}>
        <Typography
          component={component}
          sx={{
            fontWeight: 800,
            letterSpacing: '-0.02em',
            fontSize: 'clamp(1.75rem, 5vw, 2.75rem)',
            lineHeight: 1.1,
            mb: subtitle ? 1.5 : 0,
          }}
        >
          {title}
        </Typography>
        {subtitle && (
          <Typography variant="body1" color="text.secondary">
            {subtitle}
          </Typography>
        )}
      </Box>
    </Box>
  )
}
