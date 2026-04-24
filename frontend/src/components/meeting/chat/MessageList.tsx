import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { Alert, Box, Button, Chip, CircularProgress, Typography } from '@mui/material'
import KeyboardArrowDownIcon from '@mui/icons-material/KeyboardArrowDown'
import { MessageRow } from './MessageRow'
import type { ChatMessage, ParticipantDirectoryEntry } from '@/types/meeting'
import { AUTO_SCROLL_THRESHOLD_PX, GROUPING_WINDOW_MS } from '@/config/chat'

interface MessageListProps {
  messages: ChatMessage[]
  directory: Record<string, ParticipantDirectoryEntry>
  localUserId: string | null
  hasMore: boolean
  isLoading: boolean
  error: string | null
  onLoadOlder: () => void
  onRetry: () => void
  /** Retry a failed outgoing message. */
  onRetrySend?: (localId: string) => void
  /** Dismiss a failed outgoing message from the list. */
  onDeleteFailed?: (localId: string) => void
}

/**
 * Renders messages in chronological order with auto-scroll behaviour:
 *   - If the user is near the bottom, new messages scroll into view.
 *   - If scrolled up, a "↓ New messages" pill appears instead.
 *   - Scrolling to the top triggers a lazy fetch of older messages.
 */
export function MessageList({
  messages,
  directory,
  localUserId,
  hasMore,
  isLoading,
  error,
  onLoadOlder,
  onRetry,
  onRetrySend,
  onDeleteFailed,
}: MessageListProps) {
  const scrollRef = useRef<HTMLDivElement>(null)
  const atBottomRef = useRef(true)
  const [showPill, setShowPill] = useState(false)
  const prevLastIdRef = useRef<string | null>(null)
  const prevFirstIdRef = useRef<string | null>(null)
  const prevScrollHeightRef = useRef<number>(0)

  // Compute grouping flags once per messages list.
  const grouped = useMemo(() => {
    const out: boolean[] = new Array(messages.length).fill(false)
    for (let i = 1; i < messages.length; i++) {
      const prev = messages[i - 1]
      const curr = messages[i]
      out[i] = prev.userId === curr.userId && curr.sentAt - prev.sentAt < GROUPING_WINDOW_MS
    }
    return out
  }, [messages])

  const scrollToBottom = useCallback(() => {
    const el = scrollRef.current
    if (!el) return
    el.scrollTop = el.scrollHeight
    setShowPill(false)
  }, [])

  // Track scroll position to decide auto-scroll vs pill behaviour.
  useEffect(() => {
    const el = scrollRef.current
    if (!el) return
    const onScroll = () => {
      const distance = el.scrollHeight - el.scrollTop - el.clientHeight
      atBottomRef.current = distance < AUTO_SCROLL_THRESHOLD_PX
      if (atBottomRef.current) setShowPill(false)

      // Trigger lazy load when user reaches the top.
      if (el.scrollTop === 0 && hasMore && !isLoading) {
        prevScrollHeightRef.current = el.scrollHeight
        onLoadOlder()
      }
    }
    el.addEventListener('scroll', onScroll)
    return () => el.removeEventListener('scroll', onScroll)
  }, [hasMore, isLoading, onLoadOlder])

  // After paint: auto-scroll for new messages, or preserve offset when
  // prepending older history.
  useLayoutEffect(() => {
    const el = scrollRef.current
    if (!el || messages.length === 0) return

    const last = messages[messages.length - 1]
    const first = messages[0]
    const newLast = last.id !== prevLastIdRef.current
    const newFirst = first.id !== prevFirstIdRef.current

    if (newLast) {
      const isLocalSend = last.userId === localUserId && last.pending
      if (isLocalSend || atBottomRef.current) {
        el.scrollTop = el.scrollHeight
      } else {
        setShowPill(true)
      }
    } else if (newFirst && prevScrollHeightRef.current > 0) {
      // Older messages were prepended — maintain the user's visible offset.
      el.scrollTop = el.scrollHeight - prevScrollHeightRef.current
      prevScrollHeightRef.current = 0
    }

    prevLastIdRef.current = last.id
    prevFirstIdRef.current = first.id
  }, [messages, localUserId])

  return (
    <Box sx={{ position: 'relative', flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 }}>
      <Box
        ref={scrollRef}
        sx={{
          flex: 1,
          overflowY: 'auto',
          py: 1,
        }}
      >
        {error && (
          <Box sx={{ px: 2, pt: 1 }}>
            <Alert
              severity="error"
              action={
                <Button color="inherit" size="small" onClick={onRetry}>
                  Retry
                </Button>
              }
            >
              {error}
            </Alert>
          </Box>
        )}

        {hasMore && (
          <Box sx={{ py: 1, textAlign: 'center' }}>
            {isLoading
              ? <CircularProgress size={16} />
              : <Button size="small" onClick={onLoadOlder}>Load older messages</Button>}
          </Box>
        )}

        {!hasMore && !error && messages.length === 0 && !isLoading && (
          <Box sx={{ p: 3, textAlign: 'center' }}>
            <Typography variant="body2" color="text.secondary">
              No messages yet. Say hi.
            </Typography>
          </Box>
        )}

        {messages.map((m, i) => (
          <MessageRow
            key={m.id}
            message={m}
            sender={directory[m.userId] ?? null}
            grouped={grouped[i]}
            onRetry={onRetrySend}
            onDelete={onDeleteFailed}
          />
        ))}
      </Box>

      {showPill && (
        <Chip
          icon={<KeyboardArrowDownIcon />}
          label="New messages"
          onClick={scrollToBottom}
          color="primary"
          size="small"
          sx={{
            position: 'absolute',
            bottom: 8,
            left: '50%',
            transform: 'translateX(-50%)',
            zIndex: 2,
            cursor: 'pointer',
            boxShadow: 3,
          }}
        />
      )}
    </Box>
  )
}
