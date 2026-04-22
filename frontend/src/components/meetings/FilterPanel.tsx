import { Popover, Box, Typography, List, ListItemButton } from '@mui/material'
import type { MeetingStatusFilter } from '@/types/meeting'

const STATUS_OPTIONS: { value: MeetingStatusFilter | undefined; label: string }[] = [
  { value: undefined, label: 'All statuses' },
  { value: 'in_progress', label: 'in progress' },
  { value: 'complete', label: 'complete' },
  { value: 'processing', label: 'processing' },
  { value: 'failed', label: 'failed' },
]

interface Props {
  anchorEl: HTMLElement | null
  onClose: () => void
  status: MeetingStatusFilter | undefined
  onStatusChange: (value: MeetingStatusFilter | undefined) => void
}

export function FilterPanel({ anchorEl, onClose, status, onStatusChange }: Props) {
  return (
    <Popover
      open={Boolean(anchorEl)}
      anchorEl={anchorEl}
      onClose={onClose}
      anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
      transformOrigin={{ vertical: 'top', horizontal: 'right' }}
      slotProps={{ paper: { sx: { bgcolor: '#1e1e2e', border: '1px solid', borderColor: 'divider', minWidth: 200 } } }}
    >
      <Box sx={{ px: 2, pt: 2, pb: 1 }}>
        <Typography variant="overline" fontSize={10} color="text.secondary" letterSpacing="0.1em">
          STATUS
        </Typography>
      </Box>
      <List dense disablePadding sx={{ pb: 1 }}>
        {STATUS_OPTIONS.map((opt) => {
          const isActive = opt.value === status
          return (
            <ListItemButton
              key={opt.label}
              selected={isActive}
              onClick={() => {
                onStatusChange(opt.value)
                onClose()
              }}
              sx={{
                px: 2,
                py: 0.75,
                fontSize: 14,
                '&.Mui-selected': { bgcolor: 'action.selected' },
                '&.Mui-selected:hover': { bgcolor: 'action.hover' },
              }}
            >
              {opt.label}
            </ListItemButton>
          )
        })}
      </List>
    </Popover>
  )
}
