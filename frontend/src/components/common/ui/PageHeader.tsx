import { useEffect, type ReactNode } from 'react'
import { Box, Stack, Typography, type SxProps, type Theme } from '@mui/material'

interface PageHeaderProps {
  /** Heading content. Accepts a string (used for document.title too) or a
   *  ReactNode when inline accents like <GradientText> are needed. Pass
   *  `documentTitleText` alongside a ReactNode title so the browser tab
   *  still gets a plain string. */
  title: ReactNode
  subtitle?: string
  /** Right-aligned slot for primary actions (buttons, icon buttons). */
  actions?: ReactNode
  /** Optional eyebrow label above the title (e.g. "Organization"). */
  eyebrow?: string
  /** When true, syncs `document.title` to "<title> — Rekall". Default true. */
  documentTitle?: boolean
  /** Explicit document-title text to use when `title` is a ReactNode. */
  documentTitleText?: string
  sx?: SxProps<Theme>
}

/**
 * Page-level header for authenticated-shell pages. Not to be confused with
 * HeroHeader, which is for landing/auth surfaces. This component is
 * left-aligned, moderately sized, and slots actions on the right.
 */
export function PageHeader({
  title,
  subtitle,
  actions,
  eyebrow,
  documentTitle = true,
  documentTitleText,
  sx,
}: PageHeaderProps) {
  useEffect(() => {
    if (!documentTitle) return
    const resolved = documentTitleText ?? (typeof title === 'string' ? title : 'Rekall')
    document.title = `${resolved} — Rekall`
  }, [title, documentTitle, documentTitleText])

  return (
    <Box
      sx={[
        {
          display: 'flex',
          flexDirection: { xs: 'column', sm: 'row' },
          alignItems: { xs: 'flex-start', sm: 'center' },
          justifyContent: 'space-between',
          gap: { xs: 2, sm: 3 },
          mb: { xs: 3, sm: 4 },
          pb: 2,
          borderBottom: '1px solid',
          borderColor: 'divider',
        },
        ...(Array.isArray(sx) ? sx : [sx]),
      ]}
    >
      <Box sx={{ minWidth: 0 }}>
        {eyebrow && (
          <Typography
            variant="overline"
            sx={{
              color: 'text.secondary',
              letterSpacing: '0.1em',
              fontWeight: 600,
              fontSize: '0.7rem',
              display: 'block',
              mb: 0.5,
            }}
          >
            {eyebrow}
          </Typography>
        )}
        <Typography
          component="h1"
          sx={{
            fontSize: { xs: '1.5rem', sm: '1.75rem' },
            fontWeight: 700,
            letterSpacing: '-0.015em',
            lineHeight: 1.2,
            color: 'text.primary',
          }}
        >
          {title}
        </Typography>
        {subtitle && (
          <Typography
            variant="body2"
            color="text.secondary"
            sx={{ mt: 0.75, maxWidth: 620 }}
          >
            {subtitle}
          </Typography>
        )}
      </Box>

      {actions && (
        <Stack
          direction="row"
          spacing={1.25}
          alignItems="center"
          sx={{ flexShrink: 0, flexWrap: 'wrap' }}
        >
          {actions}
        </Stack>
      )}
    </Box>
  )
}
