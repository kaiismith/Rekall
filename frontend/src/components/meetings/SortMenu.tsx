import { Menu, MenuItem, ListItemIcon, Typography } from '@mui/material'
import CheckIcon from '@mui/icons-material/Check'
import type { MeetingSortKey } from '@/types/meeting'

const SORT_OPTIONS: { value: MeetingSortKey; label: string }[] = [
  { value: 'created_at_desc', label: 'Newest first' },
  { value: 'created_at_asc', label: 'Oldest first' },
  { value: 'duration_desc', label: 'Longest first' },
  { value: 'duration_asc', label: 'Shortest first' },
  { value: 'title_asc', label: 'Title A → Z' },
  { value: 'title_desc', label: 'Title Z → A' },
]

interface Props {
  anchorEl: HTMLElement | null
  onClose: () => void
  sort: MeetingSortKey
  onSortChange: (value: MeetingSortKey) => void
}

export function SortMenu({ anchorEl, onClose, sort, onSortChange }: Props) {
  return (
    <Menu
      open={Boolean(anchorEl)}
      anchorEl={anchorEl}
      onClose={onClose}
      anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
      transformOrigin={{ vertical: 'top', horizontal: 'right' }}
      slotProps={{ paper: { sx: { bgcolor: '#1e1e2e', border: '1px solid', borderColor: 'divider', minWidth: 200 } } }}
    >
      {SORT_OPTIONS.map((opt) => {
        const isActive = opt.value === sort
        return (
          <MenuItem
            key={opt.value}
            selected={isActive}
            onClick={() => {
              onSortChange(opt.value)
              onClose()
            }}
            sx={{ fontSize: 14, gap: 1 }}
          >
            <ListItemIcon sx={{ minWidth: 20 }}>
              {isActive ? <CheckIcon fontSize="small" sx={{ color: 'primary.main' }} /> : null}
            </ListItemIcon>
            <Typography fontSize={14} fontWeight={isActive ? 600 : 400}>
              {opt.label}
            </Typography>
          </MenuItem>
        )
      })}
    </Menu>
  )
}
