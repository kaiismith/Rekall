import { Box, Typography, type SxProps, type Theme } from '@mui/material'
import { Fragment } from 'react'
import { KeyChip } from './KeyChip'

interface ShortcutHintProps {
  /** Leading label, e.g. "Shortcut:". */
  label?: string
  /** Ordered list of key glyphs — each becomes a KeyChip. */
  keys: string[]
  sx?: SxProps<Theme>
  className?: string
}

export function ShortcutHint({
  label = 'Shortcut:',
  keys,
  sx,
  className,
}: ShortcutHintProps) {
  return (
    <Box
      className={className}
      aria-label={`Keyboard shortcut: ${keys.join(' plus ')}`}
      sx={[
        {
          display: { xs: 'none', sm: 'flex' },
          alignItems: 'center',
          gap: 1,
          mt: 1,
          justifyContent: 'center',
        },
        ...(Array.isArray(sx) ? sx : [sx]),
      ]}
    >
      <Typography variant="caption" color="text.secondary">
        {label}
      </Typography>
      {keys.map((k, i) => (
        <Fragment key={`${k}-${i}`}>
          <KeyChip>{k}</KeyChip>
          {i < keys.length - 1 && (
            <Typography variant="caption" color="text.secondary" aria-hidden>
              +
            </Typography>
          )}
        </Fragment>
      ))}
    </Box>
  )
}
