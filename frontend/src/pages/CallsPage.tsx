import { useQuery } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
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
import { EmptyState, PageHeader, ScopeBadge, ScopePicker } from '@/components/common/ui'
import { callService } from '@/services/callService'
import { usePagination } from '@/hooks/usePagination'
import { formatDateTime, formatDuration } from '@/utils'
import { CALL_STATUS_CONFIG } from '@/constants'
import type { Call } from '@/types/call'
import type { Scope } from '@/types/scope'
import { parseScopeFromUrl, scopeToUrlParam, scopesEqual } from '@/utils/scope'

const COLUMNS = ['Title', 'Scope', 'Status', 'Duration', 'Created']

function callToScope(c: Call): Scope {
  if (c.scope_type === 'organization' && c.scope_id) {
    return { type: 'organization', id: c.scope_id }
  }
  return { type: 'open' }
}

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

export function CallsPage() {
  const { page, perPage, setPage, setPerPage } = usePagination()
  const [searchParams, setSearchParams] = useSearchParams()
  const scope = parseScopeFromUrl(searchParams)

  const setScope = (next: Scope | null) => {
    if (scopesEqual(next, scope)) return
    setSearchParams((prev) => {
      const out = new URLSearchParams(prev)
      if (next === null) out.delete('scope')
      else out.set('scope', scopeToUrlParam(next))
      return out
    })
  }

  const scopeKey =
    scope === null
      ? 'none'
      : scope.type === 'open'
        ? 'open'
        : scope.type === 'organization'
          ? `org:${scope.id}`
          : `dept:${scope.orgId}:${scope.id}`

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['calls', { page, per_page: perPage, scope: scopeKey }],
    queryFn: () => callService.list({ page, per_page: perPage }, scope),
  })

  const calls = data?.data ?? []
  const total = data?.meta.total ?? 0

  const showEmpty = !isLoading && !isError && calls.length === 0
  const showTable = !showEmpty

  return (
    <Box>
      <PageHeader
        title="Calls"
        subtitle="Browse and manage all recorded conversations."
        actions={<ScopePicker value={scope} onChange={setScope} />}
      />

      {showEmpty ? (
        <EmptyState
          icon={<PhoneMissedOutlinedIcon />}
          title={scope ? 'No calls match this scope' : 'No calls yet'}
          description={
            scope
              ? 'Try clearing the scope filter to see all calls you can access.'
              : 'Once calls are ingested through the Rekall pipeline, they will appear here with transcripts and insights.'
          }
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

                {isError && (
                  <TableRow>
                    <TableCell colSpan={COLUMNS.length}>
                      <Box py={4} textAlign="center">
                        <Typography variant="body2" color="error.main">
                          {error instanceof Error ? error.message : 'Failed to load calls'}
                        </Typography>
                      </Box>
                    </TableCell>
                  </TableRow>
                )}

                {showTable && calls.map((call) => (
                  <TableRow
                    key={call.id}
                    hover
                    sx={{
                      cursor: 'pointer',
                      '&:last-child td': { borderBottom: 0 },
                    }}
                  >
                    <TableCell>
                      <Typography variant="body2" fontWeight={500} color="text.primary">
                        {call.title}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <ScopeBadge scope={callToScope(call)} />
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
