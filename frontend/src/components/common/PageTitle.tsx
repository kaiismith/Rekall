import { useEffect } from 'react'
import { Typography, Box, type TypographyProps } from '@mui/material'

interface PageTitleProps extends TypographyProps {
  title: string
  subtitle?: string
  documentTitle?: boolean
}

/**
 * Renders a consistent page heading and optionally updates the browser tab title.
 */
export function PageTitle({ title, subtitle, documentTitle = true, ...typographyProps }: PageTitleProps) {
  useEffect(() => {
    if (documentTitle) {
      document.title = `${title} — Rekall`
    }
  }, [title, documentTitle])

  return (
    <Box mb={3}>
      <Typography variant="h5" fontWeight={700} color="text.primary" {...typographyProps}>
        {title}
      </Typography>
      {subtitle && (
        <Typography variant="body2" color="text.secondary" mt={0.5}>
          {subtitle}
        </Typography>
      )}
    </Box>
  )
}
