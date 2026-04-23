import { Badge, IconButton, Tooltip } from '@mui/material'
import ChatBubbleOutlineIcon from '@mui/icons-material/ChatBubbleOutline'
import ChatBubbleIcon from '@mui/icons-material/ChatBubble'

interface ChatButtonProps {
  unreadCount: number
  isOpen: boolean
  onToggle: () => void
}

export function ChatButton({ unreadCount, isOpen, onToggle }: ChatButtonProps) {
  const label = isOpen
    ? 'Close chat'
    : unreadCount > 0
      ? `Open chat, ${unreadCount} unread ${unreadCount === 1 ? 'message' : 'messages'}`
      : 'Open chat'

  return (
    <Tooltip title={label}>
      <Badge
        badgeContent={unreadCount}
        color="error"
        overlap="circular"
        max={99}
        invisible={unreadCount === 0}
      >
        <IconButton
          onClick={onToggle}
          size="medium"
          aria-label={label}
          sx={{
            bgcolor: isOpen ? 'primary.main' : 'action.selected',
            color: 'white',
            '&:hover': { bgcolor: isOpen ? 'primary.dark' : 'action.hover' },
          }}
        >
          {isOpen ? <ChatBubbleIcon /> : <ChatBubbleOutlineIcon />}
        </IconButton>
      </Badge>
    </Tooltip>
  )
}
