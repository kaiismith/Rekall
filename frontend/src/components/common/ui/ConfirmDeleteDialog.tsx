import { useEffect, useRef, useState } from 'react'
import {
  Alert,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import WarningAmberRoundedIcon from '@mui/icons-material/WarningAmberRounded'
import { tokens } from '@/theme'

interface ConfirmDeleteDialogProps {
  open: boolean
  onClose: () => void
  onConfirm: () => void
  /** Title shown at the top of the dialog (e.g. "Delete organization"). */
  title: string
  /** Short human-readable name of the entity being deleted. Displayed in body copy. */
  entityName: string
  /**
   * The exact string the user must type to enable the confirm button. Should be
   * something unique and stable (e.g. a slug, a meeting code) — NOT the
   * display name, which may collide across entities.
   */
  confirmationValue: string
  /**
   * Optional label that tells the user WHAT they're typing. Defaults to
   * "slug" — override if the unique identifier is different (e.g. "code").
   */
  confirmationLabel?: string
  /** Freeform description shown above the confirmation input. */
  description?: string
  /** List of bullet-point consequences shown as a warning. */
  consequences?: string[]
  /** Label for the confirm button. Defaults to "Delete". */
  confirmLabel?: string
  /** When true, the confirm button shows a loading/disabled state. */
  loading?: boolean
  /** Optional error to display inside the dialog (e.g. from a failed mutation). */
  error?: string | null
}

/**
 * Destructive-confirmation dialog with type-to-confirm gating. Use this
 * instead of window.confirm for any irreversible action (delete org, delete
 * meeting, revoke API key). The user must type the supplied confirmation
 * value — typically the entity's unique slug or code — before the confirm
 * button activates, which prevents both accidental clicks and confusion
 * across similarly-named records.
 */
export function ConfirmDeleteDialog({
  open,
  onClose,
  onConfirm,
  title,
  entityName,
  confirmationValue,
  confirmationLabel = 'slug',
  description,
  consequences,
  confirmLabel = 'Delete',
  loading = false,
  error = null,
}: ConfirmDeleteDialogProps) {
  const [typed, setTyped] = useState('')
  const inputRef = useRef<HTMLInputElement | null>(null)

  useEffect(() => {
    if (open) {
      setTyped('')
      // Small delay so the dialog mount animation completes before focus.
      const t = setTimeout(() => inputRef.current?.focus(), 80)
      return () => clearTimeout(t)
    }
  }, [open])

  const matches = typed === confirmationValue
  const canConfirm = matches && !loading

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && canConfirm) {
      e.preventDefault()
      onConfirm()
    }
  }

  return (
    <Dialog
      open={open}
      onClose={loading ? undefined : onClose}
      maxWidth="xs"
      fullWidth
      PaperProps={{
        sx: {
          borderRadius: '14px',
          border: '1px solid rgba(239,68,68,0.25)',
        },
      }}
    >
      <DialogTitle sx={{ pb: 1.5, pt: 3 }}>
        <Stack direction="row" spacing={1.5} alignItems="center">
          <Box
            sx={{
              width: 36,
              height: 36,
              borderRadius: '10px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              bgcolor: 'rgba(239,68,68,0.12)',
              color: '#fca5a5',
              flexShrink: 0,
            }}
          >
            <WarningAmberRoundedIcon fontSize="small" />
          </Box>
          <Typography
            component="span"
            sx={{ fontWeight: 600, fontSize: '1.0625rem', letterSpacing: '-0.01em' }}
          >
            {title}
          </Typography>
        </Stack>
      </DialogTitle>

      <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: '8px !important' }}>
        {description && (
          <Typography variant="body2" color="text.secondary">
            {description}
          </Typography>
        )}

        {consequences && consequences.length > 0 && (
          <Alert
            severity="warning"
            icon={false}
            sx={{ '& .MuiAlert-message': { width: '100%' } }}
          >
            <Typography variant="caption" sx={{ fontWeight: 600, display: 'block', mb: 0.5 }}>
              This will permanently:
            </Typography>
            <Box component="ul" sx={{ m: 0, pl: 2.5, fontSize: '0.8125rem' }}>
              {consequences.map((c) => (
                <li key={c}>{c}</li>
              ))}
            </Box>
          </Alert>
        )}

        <Box>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
            To confirm, type the {confirmationLabel}{' '}
            <Box
              component="code"
              sx={{
                fontFamily: tokens.fonts.mono,
                fontSize: '0.8125rem',
                px: 0.75,
                py: 0.25,
                borderRadius: '4px',
                bgcolor: 'rgba(255,255,255,0.05)',
                border: '1px solid rgba(255,255,255,0.08)',
                color: 'text.primary',
              }}
            >
              {confirmationValue}
            </Box>{' '}
            below.
          </Typography>

          <TextField
            inputRef={inputRef}
            value={typed}
            onChange={(e) => setTyped(e.target.value)}
            onKeyDown={handleKeyDown}
            fullWidth
            placeholder={confirmationValue}
            autoComplete="off"
            spellCheck={false}
            error={typed.length > 0 && !matches}
            inputProps={{
              'aria-label': `Type ${confirmationLabel} to confirm`,
              style: { fontFamily: tokens.fonts.mono, fontSize: '0.9375rem' },
            }}
          />
        </Box>

        {error && <Alert severity="error">{error}</Alert>}

        <Typography variant="caption" color="text.secondary">
          Deleting <strong>{entityName}</strong> is permanent and cannot be undone.
        </Typography>
      </DialogContent>

      <DialogActions sx={{ px: 3, pb: 2.5, pt: 1 }}>
        <Button onClick={onClose} disabled={loading} color="inherit">
          Cancel
        </Button>
        <Button
          onClick={onConfirm}
          disabled={!canConfirm}
          variant="contained"
          color="error"
          sx={{
            minWidth: 96,
            boxShadow: 'none',
            '&:hover': { boxShadow: '0 0 0 3px rgba(239,68,68,0.15)' },
          }}
        >
          {loading ? 'Deleting…' : confirmLabel}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
