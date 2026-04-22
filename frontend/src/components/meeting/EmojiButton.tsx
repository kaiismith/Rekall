import { useRef, useState } from 'react'
import { Box, IconButton, Paper, Popover, Tooltip } from '@mui/material'
import EmojiEmotionsIcon from '@mui/icons-material/EmojiEmotions'
import { EMOJI_LIST } from '@/config/meetingControls'

interface EmojiButtonProps {
  onSend: (emoji: string) => void
}

export function EmojiButton({ onSend }: EmojiButtonProps) {
  const [open, setOpen] = useState(false)
  const anchorRef = useRef<HTMLButtonElement>(null)

  const handleSelect = (emoji: string) => {
    onSend(emoji)
    setOpen(false)
  }

  return (
    <>
      <Tooltip title="Send reaction">
        <IconButton
          ref={anchorRef}
          onClick={() => setOpen((v) => !v)}
          size="medium"
          sx={{ bgcolor: 'action.selected', color: 'white', '&:hover': { bgcolor: 'action.hover' } }}
        >
          <EmojiEmotionsIcon />
        </IconButton>
      </Tooltip>

      <Popover
        open={open}
        anchorEl={anchorRef.current}
        onClose={() => setOpen(false)}
        anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
        transformOrigin={{ vertical: 'bottom', horizontal: 'center' }}
        PaperProps={{ sx: { bgcolor: 'background.paper', borderRadius: 2 } }}
      >
        <Paper elevation={4} sx={{ p: 1 }}>
          <Box sx={{ display: 'flex', gap: 0.5 }}>
            {EMOJI_LIST.map((emoji) => (
              <IconButton
                key={emoji}
                onClick={() => handleSelect(emoji)}
                size="small"
                sx={{ fontSize: '1.5rem', borderRadius: 1, lineHeight: 1 }}
              >
                {emoji}
              </IconButton>
            ))}
          </Box>
        </Paper>
      </Popover>
    </>
  )
}
