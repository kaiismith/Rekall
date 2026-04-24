import { Avatar, Box, Button, Stack, Typography } from '@mui/material'
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline'
import { linkify } from '@/utils/linkify'
import type { ChatMessage, ParticipantDirectoryEntry } from '@/types/meeting'

interface MessageRowProps {
  message: ChatMessage
  sender: ParticipantDirectoryEntry | null
  /** When true, avatar + name are omitted (message is grouped under its predecessor). */
  grouped: boolean
  onRetry?: (localId: string) => void
  onDelete?: (localId: string) => void
}

function formatTimestamp(ms: number): string {
  const delta = Date.now() - ms
  if (delta < 60_000) return 'just now'
  if (delta < 60 * 60_000) return `${Math.floor(delta / 60_000)}m ago`
  const d = new Date(ms)
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

export function MessageRow({ message, sender, grouped, onRetry, onDelete }: MessageRowProps) {
  const displayName = sender?.full_name ?? 'User'
  const initials = sender?.initials ?? '?'

  return (
    <Box
      data-testid="chat-message-row"
      data-pending={message.pending ? 'true' : 'false'}
      data-failed={message.failed ? 'true' : 'false'}
      sx={{
        display: 'flex',
        gap: 1.25,
        px: 2,
        py: grouped ? 0.25 : 0.75,
        opacity: message.pending ? 0.6 : 1,
        '&:hover': { bgcolor: 'action.hover' },
      }}
    >
      {/* Avatar slot — width reserved even when hidden so message body aligns. */}
      <Box sx={{ width: 32, flexShrink: 0 }}>
        {!grouped && (
          <Avatar sx={{ width: 28, height: 28, fontSize: '0.75rem', bgcolor: 'primary.dark' }}>
            {initials}
          </Avatar>
        )}
      </Box>

      <Box sx={{ flex: 1, minWidth: 0 }}>
        {!grouped && (
          <Stack direction="row" spacing={1} alignItems="baseline" mb={0.25}>
            <Typography variant="body2" fontWeight={600} noWrap>
              {displayName}
            </Typography>
            <Typography variant="caption" color="text.secondary" sx={{ flexShrink: 0 }}>
              {formatTimestamp(message.sentAt)}
            </Typography>
          </Stack>
        )}
        <Typography
          variant="body2"
          component="div"
          sx={{
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-word',
            '& a': { color: 'primary.main', textDecoration: 'underline' },
          }}
        >
          {linkify(message.body)}
        </Typography>

        {message.failed && (
          <Stack direction="row" spacing={1} alignItems="center" mt={0.5}>
            <ErrorOutlineIcon fontSize="inherit" sx={{ color: 'error.main' }} />
            <Typography variant="caption" color="error.main">
              Not delivered
            </Typography>
            {onRetry && (
              <Button
                size="small"
                variant="text"
                onClick={() => onRetry(message.id)}
                sx={{ minWidth: 'auto', p: 0, fontSize: '0.75rem' }}
              >
                Retry
              </Button>
            )}
            {onDelete && (
              <Button
                size="small"
                variant="text"
                color="inherit"
                onClick={() => onDelete(message.id)}
                sx={{ minWidth: 'auto', p: 0, fontSize: '0.75rem', color: 'text.secondary' }}
              >
                Dismiss
              </Button>
            )}
          </Stack>
        )}
      </Box>
    </Box>
  )
}
