import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Badge, Box, CircularProgress, IconButton, Stack, Tooltip } from '@mui/material'
import FilterListIcon from '@mui/icons-material/FilterList'
import SortIcon from '@mui/icons-material/Sort'
import AddIcon from '@mui/icons-material/Add'
import VideoCallOutlinedIcon from '@mui/icons-material/VideoCallOutlined'
import { useMeetingsList } from '@/hooks/useMeetingsList'
import { MeetingCard } from '@/components/meetings/MeetingCard'
import { FilterPanel } from '@/components/meetings/FilterPanel'
import { SortMenu } from '@/components/meetings/SortMenu'
import { EmptyState, GradientButton, PageHeader, ScopePicker } from '@/components/common/ui'
import { ROUTES } from '@/constants'

const iconBtnSx = {
  bgcolor: 'rgba(255,255,255,0.03)',
  border: '1px solid rgba(255,255,255,0.06)',
  borderRadius: '8px',
  width: 36,
  height: 36,
  '&:hover': { bgcolor: 'rgba(255,255,255,0.06)' },
}

/**
 * Records list — every meeting the caller hosted or attended, surfaced as a
 * Record. Cards navigate to /records/:code (the detail page with the stored
 * transcript) instead of /meeting/:code (the live WebRTC room).
 */
export function RecordsPage() {
  const navigate = useNavigate()
  const {
    meetings,
    isLoading,
    status,
    sort,
    scope,
    setStatus,
    setSort,
    setScope,
    activeFilterCount,
  } = useMeetingsList()

  const [filterAnchor, setFilterAnchor] = useState<HTMLElement | null>(null)
  const [sortAnchor, setSortAnchor] = useState<HTMLElement | null>(null)

  const actions = (
    <>
      <ScopePicker value={scope} onChange={setScope} />

      <Tooltip title="Filter">
        <Badge badgeContent={activeFilterCount} color="primary" overlap="circular">
          <IconButton
            size="small"
            onClick={(e) => setFilterAnchor(e.currentTarget)}
            sx={{
              ...iconBtnSx,
              color: activeFilterCount > 0 ? 'primary.light' : 'text.secondary',
            }}
          >
            <FilterListIcon fontSize="small" />
          </IconButton>
        </Badge>
      </Tooltip>

      <Tooltip title="Sort">
        <IconButton
          size="small"
          onClick={(e) => setSortAnchor(e.currentTarget)}
          sx={{
            ...iconBtnSx,
            color: sort !== 'created_at_desc' ? 'primary.light' : 'text.secondary',
          }}
        >
          <SortIcon fontSize="small" />
        </IconButton>
      </Tooltip>

      <GradientButton
        size="small"
        fullWidth={false}
        startIcon={<AddIcon />}
        onClick={() => navigate(ROUTES.NEW_RECORD)}
      >
        New Meeting
      </GradientButton>
    </>
  )

  return (
    <Box sx={{ maxWidth: 960, mx: 'auto' }}>
      <PageHeader
        title="Your Records"
        subtitle="Browse past meetings and read their stored transcripts."
        actions={actions}
      />

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

      {isLoading ? (
        <Box display="flex" justifyContent="center" py={8}>
          <CircularProgress />
        </Box>
      ) : meetings.length === 0 ? (
        <EmptyState
          icon={<VideoCallOutlinedIcon />}
          title={status || scope ? 'No records match this filter' : 'No records yet'}
          description={
            status || scope
              ? 'Try clearing the active filters or adjusting your sort order.'
              : 'Start a meeting to capture conversations, transcripts, and highlights.'
          }
          action={
            !status &&
            !scope && (
              <GradientButton
                fullWidth={false}
                startIcon={<AddIcon />}
                onClick={() => navigate(ROUTES.NEW_RECORD)}
              >
                Start a meeting
              </GradientButton>
            )
          }
        />
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
