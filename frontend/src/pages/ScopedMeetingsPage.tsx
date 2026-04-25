import { useNavigate, useParams } from 'react-router-dom'
import {
  Badge,
  Box,
  CircularProgress,
  IconButton,
  Stack,
  Tooltip,
} from '@mui/material'
import FilterListIcon from '@mui/icons-material/FilterList'
import SortIcon from '@mui/icons-material/Sort'
import AddIcon from '@mui/icons-material/Add'
import VideoCallOutlinedIcon from '@mui/icons-material/VideoCallOutlined'
import { useState } from 'react'

import { isUuid, ROUTES } from '@/constants'
import type { Scope } from '@/types/scope'
import { scopeToUrlParam } from '@/utils/scope'
import { useMeetingsList } from '@/hooks/useMeetingsList'
import { MeetingCard } from '@/components/meetings/MeetingCard'
import { FilterPanel } from '@/components/meetings/FilterPanel'
import { SortMenu } from '@/components/meetings/SortMenu'
import {
  AccessDeniedState,
  EmptyState,
  GradientButton,
  PageHeader,
  ScopeBreadcrumb,
} from '@/components/common/ui'

const iconBtnSx = {
  bgcolor: 'rgba(255,255,255,0.03)',
  border: '1px solid rgba(255,255,255,0.06)',
  borderRadius: '8px',
  width: 36,
  height: 36,
  '&:hover': { bgcolor: 'rgba(255,255,255,0.06)' },
}

interface ScopedMeetingsPageProps {
  /**
   * When provided, the scope is fixed by the caller (e.g., embedded inside the
   * Org Detail page's Meetings tab). When omitted, the route params determine
   * the scope and the breadcrumb + page header are rendered.
   */
  scope?: Scope
  /** Hide the breadcrumb + hero (used when embedded in another page). */
  embedded?: boolean
}

/**
 * Renders the Meetings list pre-filtered to a given organization or department.
 *
 * Two mount paths:
 *   1. Standalone route — `/organizations/:id/meetings` or
 *      `/organizations/:orgId/departments/:deptId/meetings`. The scope is read
 *      from route params; breadcrumb + chrome are rendered.
 *   2. Embedded — used inside `OrgDetailPage`'s "Meetings" tab. Scope passed
 *      explicitly, breadcrumb suppressed, header suppressed.
 */
export function ScopedMeetingsPage({ scope: scopeProp, embedded }: ScopedMeetingsPageProps) {
  const navigate = useNavigate()
  const params = useParams<{ id?: string; orgId?: string; deptId?: string }>()

  const routeScope = resolveRouteScope(params)
  const scope = scopeProp ?? routeScope

  if (!scope) return <AccessDeniedState />

  return <ScopedMeetingsBody scope={scope} embedded={!!embedded} navigate={navigate} />
}

function resolveRouteScope(params: {
  id?: string
  orgId?: string
  deptId?: string
}): Scope | null {
  if (params.deptId && params.orgId) {
    if (!isUuid(params.deptId) || !isUuid(params.orgId)) return null
    return { type: 'department', id: params.deptId, orgId: params.orgId }
  }
  const orgId = params.id ?? params.orgId
  if (!orgId || !isUuid(orgId)) return null
  return { type: 'organization', id: orgId }
}

function ScopedMeetingsBody({
  scope,
  embedded,
  navigate,
}: {
  scope: Scope
  embedded: boolean
  navigate: ReturnType<typeof useNavigate>
}) {
  const { meetings, isLoading, isError, status, sort, setStatus, setSort, activeFilterCount } =
    useMeetingsList({ forcedScope: scope })

  const [filterAnchor, setFilterAnchor] = useState<HTMLElement | null>(null)
  const [sortAnchor, setSortAnchor] = useState<HTMLElement | null>(null)

  // Forbidden / not-found from the scoped list endpoint surfaces as an error;
  // the membership check returns 403 for non-members.
  if (isError) return <AccessDeniedState />

  const newMeetingHref = `${ROUTES.NEW_MEETING}?scope=${encodeURIComponent(scopeToUrlParam(scope))}`

  const actions = (
    <>
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
        onClick={() => navigate(newMeetingHref)}
      >
        New Meeting
      </GradientButton>
    </>
  )

  return (
    <Box sx={{ maxWidth: 960, mx: 'auto' }}>
      {!embedded && <ScopeBreadcrumb />}
      {!embedded && (
        <PageHeader
          title={scope.type === 'department' ? 'Department Meetings' : 'Organization Meetings'}
          subtitle="Meetings scoped to this team. Open items are not shown here."
          actions={actions}
        />
      )}
      {embedded && (
        <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 1, mb: 2 }}>{actions}</Box>
      )}

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
          title={status ? 'No meetings match this filter' : 'No meetings in this scope yet'}
          description={
            status
              ? 'Try clearing the active filter or adjusting your sort order.'
              : 'Create the first meeting for this team to get started.'
          }
          action={
            !status && (
              <GradientButton
                fullWidth={false}
                startIcon={<AddIcon />}
                onClick={() => navigate(newMeetingHref)}
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
