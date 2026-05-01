import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Badge, Box, Button, CircularProgress, IconButton, Stack, Tooltip } from '@mui/material'
import FilterListIcon from '@mui/icons-material/FilterList'
import SortIcon from '@mui/icons-material/Sort'
import AddIcon from '@mui/icons-material/Add'
import VideoCallOutlinedIcon from '@mui/icons-material/VideoCallOutlined'
import { useMeetingsList } from '@/hooks/useMeetingsList'
import { MeetingCard } from '@/components/meetings/MeetingCard'
import { FilterPanel } from '@/components/meetings/FilterPanel'
import { SortMenu } from '@/components/meetings/SortMenu'
import { EmptyState, GradientButton, PageHeader, ScopePicker } from '@/components/common/ui'
import { RecordDetail, RecordsEmptyState } from '@/components/records/RecordDetail'
import { ROUTES } from '@/constants'
import type { Meeting } from '@/types/meeting'

const iconBtnSx = {
  bgcolor: 'rgba(255,255,255,0.03)',
  border: '1px solid rgba(255,255,255,0.06)',
  borderRadius: '8px',
  width: 36,
  height: 36,
  '&:hover': { bgcolor: 'rgba(255,255,255,0.06)' },
}

/**
 * Records page — two-pane layout. Left: list of records (always visible).
 * Right: either the "Intelligence Ready" empty state (when no record is
 * selected, i.e. URL is /records) or the selected record's detail view (when
 * URL is /records/:code).
 */
export function RecordsPage() {
  const navigate = useNavigate()
  const { code: selectedCode } = useParams<{ code: string }>()

  const {
    meetings,
    isLoading,
    isFetchingNextPage,
    hasNextPage,
    loadMore,
    total,
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

  const remaining = Math.max(0, total - meetings.length)

  const recordLink = (m: Meeting) => `/records/${m.code}`

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
    <Box
      sx={{
        display: 'grid',
        gridTemplateColumns: { xs: '1fr', md: 'minmax(360px, 480px) 1fr' },
        gap: 3,
        alignItems: 'flex-start',
      }}
    >
      {/* ─── Left pane: list of records ───────────────────────────────────── */}
      <Box sx={{ minWidth: 0 }}>
        <PageHeader title="Your Records" actions={actions} />

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
          <Stack
            spacing={1.5}
            sx={{
              maxHeight: 'calc(100vh - 220px)',
              overflowY: 'auto',
              pr: 1,
            }}
          >
            {meetings.map((m) => (
              <Box
                key={m.id}
                sx={{
                  outline: m.code === selectedCode ? '2px solid' : 'none',
                  outlineColor: 'primary.main',
                  borderRadius: 2,
                }}
              >
                <MeetingCard meeting={m} linkTo={recordLink} />
              </Box>
            ))}
            {hasNextPage && (
              <Button
                variant="outlined"
                onClick={loadMore}
                disabled={isFetchingNextPage}
                startIcon={isFetchingNextPage ? <CircularProgress size={14} /> : undefined}
                sx={{ alignSelf: 'center', mt: 1 }}
              >
                {isFetchingNextPage
                  ? 'Loading…'
                  : `Show ${remaining > 0 ? remaining : ''} more`.trim()}
              </Button>
            )}
          </Stack>
        )}
      </Box>

      {/* ─── Right pane: detail or empty state ────────────────────────────── */}
      <Box sx={{ minWidth: 0, position: 'sticky', top: 16 }}>
        {selectedCode ? <RecordDetail code={selectedCode} /> : <RecordsEmptyState />}
      </Box>
    </Box>
  )
}
