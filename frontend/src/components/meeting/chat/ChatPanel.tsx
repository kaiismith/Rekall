import { Box, IconButton, Typography, useMediaQuery, useTheme } from '@mui/material'
import CloseIcon from '@mui/icons-material/Close'
import { MessageList } from './MessageList'
import { Composer } from './Composer'
import { CHAT_PANEL_WIDTH_PX } from '@/config/chat'
import type { ChatMessage, ParticipantDirectoryEntry } from '@/types/meeting'

interface ChatPanelProps {
  isOpen: boolean
  onClose: () => void
  messages: ChatMessage[]
  directory: Record<string, ParticipantDirectoryEntry>
  localUserId: string | null
  hasMore: boolean
  isLoading: boolean
  historyError: string | null
  sendError: string | null
  onLoadOlder: () => void
  onRetry: () => void
  onSend: (body: string) => void
  onRetrySend: (localId: string) => void
  onDeleteFailed: (localId: string) => void
  onDismissSendError: () => void
  composerDisabled: boolean
  flashKey: number
}

/**
 * Right-side chat drawer. Remains mounted when closed (visibility toggle) so
 * live messages continue to update state and history is not re-fetched on
 * every reopen.
 */
export function ChatPanel({
  isOpen,
  onClose,
  messages,
  directory,
  localUserId,
  hasMore,
  isLoading,
  historyError,
  sendError,
  onLoadOlder,
  onRetry,
  onSend,
  onRetrySend,
  onDeleteFailed,
  onDismissSendError,
  composerDisabled,
  flashKey,
}: ChatPanelProps) {
  const theme = useTheme()
  const isMobile = useMediaQuery(theme.breakpoints.down('md'))

  const handleSend = (body: string) => {
    if (sendError) onDismissSendError()
    onSend(body)
  }

  return (
    <Box
      role="region"
      aria-label="Chat panel"
      data-testid="chat-panel"
      aria-hidden={!isOpen}
      sx={{
        display: isOpen ? 'flex' : 'none',
        flexDirection: 'column',
        flexShrink: 0,
        width: isMobile ? '100%' : `${CHAT_PANEL_WIDTH_PX}px`,
        height: '100%',
        borderLeft: isMobile ? 'none' : '1px solid',
        borderColor: 'divider',
        bgcolor: 'background.paper',
        ...(isMobile && isOpen && {
          position: 'absolute',
          inset: 0,
          zIndex: 20,
        }),
      }}
    >
      <Box
        sx={{
          px: 2,
          py: 1.25,
          borderBottom: '1px solid',
          borderColor: 'divider',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          flexShrink: 0,
        }}
      >
        <Typography variant="subtitle1" fontWeight={600}>Chat</Typography>
        <IconButton size="small" onClick={onClose} aria-label="Close chat">
          <CloseIcon fontSize="small" />
        </IconButton>
      </Box>

      <MessageList
        messages={messages}
        directory={directory}
        localUserId={localUserId}
        hasMore={hasMore}
        isLoading={isLoading}
        error={historyError}
        onLoadOlder={onLoadOlder}
        onRetry={onRetry}
        onRetrySend={onRetrySend}
        onDeleteFailed={onDeleteFailed}
      />

      {sendError && (
        <Box sx={{ px: 2, py: 1, bgcolor: 'error.dark', color: 'error.contrastText' }}>
          <Typography variant="caption">{sendError}</Typography>
        </Box>
      )}

      <Composer
        disabled={composerDisabled}
        onSend={handleSend}
        flashKey={flashKey}
        onEscEmpty={onClose}
      />
    </Box>
  )
}
