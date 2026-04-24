import { Box, Stack, Typography, type SxProps, type Theme } from '@mui/material'
import type { ReactNode } from 'react'
import { tokens } from '@/theme'

interface EmptyStateProps {
  title: string
  description?: string
  icon?: ReactNode
  /** Primary call-to-action. */
  action?: ReactNode
  /** Optional secondary link/action rendered below the primary. */
  secondaryAction?: ReactNode
  sx?: SxProps<Theme>
}

/**
 * Consistent empty-state surface for list pages (no meetings, no calls,
 * no organisations). Uses the same elevated-card language as ActionCard
 * but with content centred and spaced for a "nothing here yet" moment.
 */
export function EmptyState({
  title,
  description,
  icon,
  action,
  secondaryAction,
  sx,
}: EmptyStateProps) {
  return (
    <Box
      sx={[
        {
          borderRadius: `${tokens.radii.card}px`,
          border: '1px solid rgba(255,255,255,0.06)',
          background: tokens.gradients.cardSurface,
          boxShadow: tokens.shadows.elevatedCard,
          px: { xs: 3, sm: 6 },
          py: { xs: 6, sm: 8 },
          textAlign: 'center',
        },
        ...(Array.isArray(sx) ? sx : [sx]),
      ]}
    >
      <Stack spacing={2} alignItems="center">
        {icon && (
          <Box
            sx={{
              width: 56,
              height: 56,
              borderRadius: '14px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              bgcolor: 'rgba(129,140,248,0.1)',
              color: '#a78bfa',
              mb: 1,
              '& > *': { fontSize: '1.75rem' },
            }}
          >
            {icon}
          </Box>
        )}
        <Typography
          variant="h6"
          sx={{ fontWeight: 600, color: 'text.primary', letterSpacing: '-0.01em' }}
        >
          {title}
        </Typography>
        {description && (
          <Typography
            variant="body2"
            color="text.secondary"
            sx={{ maxWidth: 420, mx: 'auto' }}
          >
            {description}
          </Typography>
        )}
        {(action || secondaryAction) && (
          <Stack spacing={1} alignItems="center" sx={{ mt: 1.5 }}>
            {action}
            {secondaryAction}
          </Stack>
        )}
      </Stack>
    </Box>
  )
}
