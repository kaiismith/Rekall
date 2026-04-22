import { useQuery } from '@tanstack/react-query'
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
import { PageTitle } from '@/components/common/PageTitle'
import { callService } from '@/services/callService'
import { usePagination } from '@/hooks/usePagination'
import { formatDateTime, formatDuration } from '@/utils'
import { CALL_STATUS_CONFIG } from '@/constants'
import type { Call } from '@/types/call'

const COLUMNS = ['Title', 'Status', 'Duration', 'Created At']

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
          <Skeleton variant="text" width="80%" />
        </TableCell>
      ))}
    </TableRow>
  )
}

export function CallsPage() {
  const { page, perPage, setPage, setPerPage } = usePagination()

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['calls', { page, per_page: perPage }],
    queryFn: () => callService.list({ page, per_page: perPage }),
  })

  const calls = data?.data ?? []
  const total = data?.meta.total ?? 0

  return (
    <>
      <PageTitle title="Calls" subtitle="Browse and manage all recorded conversations" />

      <Card>
        <TableContainer>
          <Table size="small">
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
                      <Typography variant="body2" color="error">
                        {error instanceof Error ? error.message : 'Failed to load calls'}
                      </Typography>
                    </Box>
                  </TableCell>
                </TableRow>
              )}

              {!isLoading && !isError && calls.length === 0 && (
                <TableRow>
                  <TableCell colSpan={COLUMNS.length}>
                    <Box py={6} textAlign="center">
                      <Typography variant="body2" color="text.secondary">
                        No calls yet. Calls will appear here once they are ingested.
                      </Typography>
                    </Box>
                  </TableCell>
                </TableRow>
              )}

              {calls.map((call) => (
                <TableRow key={call.id} hover sx={{ cursor: 'pointer' }}>
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
      </Card>
    </>
  )
}
