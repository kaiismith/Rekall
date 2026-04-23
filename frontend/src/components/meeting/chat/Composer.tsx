import { useEffect, useRef, useState, type KeyboardEvent } from 'react'
import { Box, IconButton, TextField, Tooltip, Typography } from '@mui/material'
import SendIcon from '@mui/icons-material/Send'
import { MAX_MESSAGE_LENGTH, CHAR_COUNTER_WARN_AT } from '@/config/chat'

interface ComposerProps {
  disabled: boolean
  onSend: (body: string) => void
  /** Incremented by the parent to flash a red outline (rate-limit feedback). */
  flashKey?: number
  /** Called when Esc is pressed and the composer is empty. */
  onEscEmpty?: () => void
}

export function Composer({ disabled, onSend, flashKey, onEscEmpty }: ComposerProps) {
  const [value, setValue] = useState('')
  const [flashing, setFlashing] = useState(false)
  const inputRef = useRef<HTMLTextAreaElement>(null)

  // Trigger red outline when flashKey ticks.
  useEffect(() => {
    if (flashKey === undefined || flashKey === 0) return
    setFlashing(true)
    const t = setTimeout(() => setFlashing(false), 300)
    return () => clearTimeout(t)
  }, [flashKey])

  const submit = () => {
    const trimmed = value.trim()
    if (!trimmed) return
    onSend(trimmed)
    setValue('')
  }

  const handleKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      submit()
      return
    }
    if (e.key === 'Escape' && value === '') {
      onEscEmpty?.()
    }
  }

  const over = value.length >= CHAR_COUNTER_WARN_AT
  const empty = value.trim().length === 0

  return (
    <Box
      sx={{
        p: 1.5,
        borderTop: '1px solid',
        borderColor: 'divider',
        display: 'flex',
        alignItems: 'flex-end',
        gap: 1,
        bgcolor: 'background.paper',
      }}
    >
      <Box sx={{ flex: 1, position: 'relative' }}>
        <TextField
          inputRef={inputRef}
          value={value}
          onChange={(e) => {
            const v = e.target.value
            setValue(v.length > MAX_MESSAGE_LENGTH ? v.slice(0, MAX_MESSAGE_LENGTH) : v)
          }}
          onKeyDown={handleKeyDown}
          placeholder={disabled ? 'Chat unavailable' : 'Send a message…'}
          multiline
          maxRows={5}
          fullWidth
          size="small"
          disabled={disabled}
          inputProps={{ maxLength: MAX_MESSAGE_LENGTH, 'aria-label': 'Chat message' }}
          sx={{
            '& .MuiOutlinedInput-root': {
              transition: 'box-shadow 0.2s, border-color 0.2s',
              ...(flashing && {
                boxShadow: '0 0 0 2px rgba(239, 68, 68, 0.5)',
                '& fieldset': { borderColor: 'error.main !important' },
              }),
            },
          }}
        />
        <Typography
          variant="caption"
          sx={{
            position: 'absolute',
            bottom: 4,
            right: 8,
            color: over ? 'error.main' : 'text.disabled',
            pointerEvents: 'none',
            fontVariantNumeric: 'tabular-nums',
          }}
        >
          {value.length}/{MAX_MESSAGE_LENGTH}
        </Typography>
      </Box>
      <Tooltip title="Send message">
        <span>
          <IconButton
            aria-label="Send message"
            onClick={submit}
            disabled={disabled || empty}
            color="primary"
          >
            <SendIcon />
          </IconButton>
        </span>
      </Tooltip>
    </Box>
  )
}
