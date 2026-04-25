import { useQuery } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import Box from '@mui/material/Box'
import Card from '@mui/material/Card'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableContainer from '@mui/material/TableContainer'
import TableHead from '@mui/material/TableHead'
import TableRow from '@mui/material/TableRow'
import Chip from '@mui/material/Chip'
import Typography from '@mui/material/Typography'
import TablePagination from '@mui/material/TablePagination'
import Skeleton from '@mui/material/Skeleton'
import PhoneMissedOutlinedIcon from '@mui/icons-material/PhoneMissedOutlined'
import {
  AccessDeniedState,
  EmptyState,
  PageHeader,
  ScopeBreadcrumb,
} from '@/components/common/ui'
import { callService } from '@/services/callService'
import { usePagination } from '@/hooks/usePagination'
import { formatDateTime, formatDuration } from '@/utils'
import { CALL_STATUS_CONFIG, isUuid } from '@/constants'
import type { Call } from '@/types/call'
import type { Scope } from '@/types/scope'

const COLUMNS = ['Title', 'Status', 'Duration', 'Created']

function StatusChip({ status }: { status: Call['status'] }) {
  const config = CALL_STATUS_CONFIG[status]
  return (
    <Chip
      label={config.label}
      color={config.color}
      size="small"
      variant="outlined"
      sx={{ fontWeight: 600, fontSize: '0.7rem' }}
    />
  )
}

function TableRowSkeleton() {
  return (
    <TableRow>
      {COLUMNS.map((col) => (
        <TableCell key={col}>
          <Skeleton variant="text" width="70%" />
        </TableCell>
      ))}
    </TableRow>
  )
}

interface ScopedCallsPageProps {
  scope?: Scope
  embedded?: boolean
}

/**
 * Calls list pre-filtered to a single org or dept. Mirrors `ScopedMeetingsPage`'s
 * mount paths: standalone (route params) or embedded (passed `scope`).
 */
export function ScopedCallsPage({ scope: scopeProp, embedded }: ScopedCallsPageProps) {
  const params = useParams<{ id?: string; orgId?: string; deptId?: string }>()
  const routeScope = resolveRouteScope(params)
  const scope = scopeProp ?? routeScope

  if (!scope) return <AccessDeniedState />

  return <ScopedCallsBody scope={scope} embedded={!!embedded} />
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

function ScopedCallsBody({ scope, embedded }: { scope: Scope; embedded: boolean }) {
  const { page, perPage, setPage, setPerPage } = usePagination()
  const scopeKey =
    scope.type === 'open'
      ? 'open'
      : scope.type === 'organization'
        ? `org:${scope.id}`
        : `dept:${scope.orgId}:${scope.id}`

  const { data, isLoading, isError } = useQuery({
    queryKey: ['calls', 'scoped', { page, per_page: perPage, scope: scopeKey }],
    queryFn: () => callService.list({ page, per_page: perPage }, scope),
  })

  const calls = data?.data ?? []
  const total = data?.meta.total ?? 0

  if (isError) return <AccessDeniedState />

  const showEmpty = !isLoading && calls.length === 0
  const showTable = !showEmpty

  return (
    <Box>
      {!embedded && <ScopeBreadcrumb />}
      {!embedded && (
        <PageHeader
          title={scope.type === 'department' ? 'Department Calls' : 'Organization Calls'}
          subtitle="Calls scoped to this team. Open items are not shown here."
        />
      )}

      {showEmpty ? (
        <EmptyState
          icon={<PhoneMissedOutlinedIcon />}
          title="No calls in this scope yet"
          description="Calls attached to this team will appear here as they are ingested."
        />
      ) : (
        <Card>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  {COLUMNS.map((col) => (
                    <TableCell key={col}>{col}</TableCell>
                  ))}
                </TableRow>
              </TableHead>
              <TableBody>
                {isLoading && Array.from({ length: 5 }).map((_, i) => <TableRowSkeleton key={i} />)}

                {showTable &&
                  calls.map((call) => (
                    <TableRow
                      key={call.id}
                      hover
                      sx={{ cursor: 'pointer', '&:last-child td': { borderBottom: 0 } }}
                    >
                      <TableCell>
                        <Typography variant="body2" fontWeight={500} color="text.primary">
                          {call.title}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <StatusChip status={call.status} />
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2" color="text.secondary">
                          {formatDuration(call.duration_sec)}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2" color="text.secondary">
                          {formatDateTime(call.created_at)}
                        </Typography>
                      </TableCell>
                    </TableRow>
                  ))}
              </TableBody>
            </Table>
          </TableContainer>

          {showTable && (
            <TablePagination
              component="div"
              count={total}
              page={page - 1}
              rowsPerPage={perPage}
              rowsPerPageOptions={[10, 20, 50]}
              onPageChange={(_, newPage) => setPage(newPage + 1)}
              onRowsPerPageChange={(e) => setPerPage(Number(e.target.value))}
              sx={{ borderTop: '1px solid', borderColor: 'divider' }}
            />
          )}
        </Card>
      )}
    </Box>
  )
}
