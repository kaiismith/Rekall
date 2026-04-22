import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Box,
  Badge,
  Button,
  CircularProgress,
  IconButton,
  Paper,
  Stack,
  Typography,
  Tooltip,
} from '@mui/material'
import FilterListIcon from '@mui/icons-material/FilterList'
import SortIcon from '@mui/icons-material/Sort'
import AddIcon from '@mui/icons-material/Add'
import { useMeetingsList } from '@/hooks/useMeetingsList'
import { MeetingCard } from '@/components/meetings/MeetingCard'
import { FilterPanel } from '@/components/meetings/FilterPanel'
import { SortMenu } from '@/components/meetings/SortMenu'

export function MeetingsPage() {
  const navigate = useNavigate()
  const { meetings, isLoading, status, sort, setStatus, setSort, activeFilterCount } =
    useMeetingsList()

  const [filterAnchor, setFilterAnchor] = useState<HTMLElement | null>(null)
  const [sortAnchor, setSortAnchor] = useState<HTMLElement | null>(null)

  return (
    <Box sx={{ p: 3, maxWidth: 680, mx: 'auto' }}>
      {/* Header */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h5" fontWeight={700}>
          Your Meetings
        </Typography>

        <Stack direction="row" spacing={1} alignItems="center">
          {/* Filter button */}
          <Tooltip title="Filter">
            <Badge badgeContent={activeFilterCount} color="primary" overlap="circular">
              <IconButton
                size="small"
                onClick={(e) => setFilterAnchor(e.currentTarget)}
                sx={{ color: activeFilterCount > 0 ? 'primary.main' : 'text.secondary' }}
              >
                <FilterListIcon fontSize="small" />
              </IconButton>
            </Badge>
          </Tooltip>

          {/* Sort button */}
          <Tooltip title="Sort">
            <IconButton
              size="small"
              onClick={(e) => setSortAnchor(e.currentTarget)}
              sx={{ color: sort !== 'created_at_desc' ? 'primary.main' : 'text.secondary' }}
            >
              <SortIcon fontSize="small" />
            </IconButton>
          </Tooltip>

          {/* New Meeting */}
          <Button
            variant="contained"
            size="small"
            startIcon={<AddIcon />}
            onClick={() => navigate('/meetings/new')}
            sx={{ ml: 0.5 }}
          >
            New Meeting
          </Button>
        </Stack>
      </Stack>

      {/* Popovers */}
      <FilterPanel
        anchorEl={filterAnchor}
        onClose={() => setFilterAnchor(null)}
        status={status}
        onStatusChange={setStatus}
      />
      <SortMenu
        anchorEl={sortAnchor}
        onClose={() => setSortAnchor(null)}
        sort={sort}
        onSortChange={setSort}
      />

      {/* Content */}
      {isLoading ? (
        <Box display="flex" justifyContent="center" mt={8}>
          <CircularProgress />
        </Box>
      ) : meetings.length === 0 ? (
        <Paper
          sx={{
            p: 6,
            textAlign: 'center',
            bgcolor: 'background.paper',
            borderRadius: 2,
          }}
        >
          <Typography color="text.secondary" mb={2}>
            {status ? 'No meetings match this filter.' : 'No meetings yet.'}
          </Typography>
          {!status && (
            <Button variant="contained" onClick={() => navigate('/meetings/new')}>
              Start a Meeting
            </Button>
          )}
        </Paper>
      ) : (
        <Stack spacing={1.5}>
          {meetings.map((m) => (
            <MeetingCard key={m.id} meeting={m} />
          ))}
        </Stack>
      )}
    </Box>
  )
}
