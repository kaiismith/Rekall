import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  MenuItem,
  Select,
  Stack,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Tabs,
  TextField,
  Typography,
} from '@mui/material'
import PersonAddIcon from '@mui/icons-material/PersonAdd'
import DeleteIcon from '@mui/icons-material/Delete'
import ArrowBackIcon from '@mui/icons-material/ArrowBack'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'

import { organizationService } from '@/services/organizationService'
import { ApiError } from '@/services/api'
import { isUuid, buildScopedRoute } from '@/constants'
import {
  AccessDeniedState,
  GradientButton,
  HeroHeader,
  ScopeBreadcrumb,
} from '@/components/common/ui'
import { ScopedMeetingsPage } from './ScopedMeetingsPage'
import { ScopedCallsPage } from './ScopedCallsPage'
import { useOrgsStore } from '@/store/orgsStore'
import { useAuthStore } from '@/store/authStore'
import { canAddDeptMember, canPromoteDeptMember } from '@/utils/permissions'
import { useStalePermissionHandler } from '@/hooks/useStalePermissionHandler'

const TABS = ['overview', 'meetings', 'calls'] as const
type Tab = (typeof TABS)[number]

function normalizeTab(value: string | null): Tab {
  return TABS.includes(value as Tab) ? (value as Tab) : 'overview'
}

export function DeptDetailPage() {
  const { orgId, deptId } = useParams<{ orgId: string; deptId: string }>()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const tab = normalizeTab(searchParams.get('tab'))

  if (!isUuid(orgId) || !isUuid(deptId)) return <AccessDeniedState />

  const setTab = (next: Tab) => {
    setSearchParams(
      (prev) => {
        const out = new URLSearchParams(prev)
        out.set('tab', next)
        return out
      },
      { replace: true },
    )
  }

  return (
    <Box>
      <ScopeBreadcrumb />

      <Box sx={{ px: { xs: 2, sm: 3 }, py: 1 }}>
        <Button
          startIcon={<ArrowBackIcon />}
          onClick={() => navigate(buildScopedRoute.org(orgId!) + '?tab=departments')}
          size="small"
          sx={{ color: 'text.secondary', textTransform: 'none' }}
        >
          Back to organization
        </Button>
      </Box>

      <DeptHero orgId={orgId!} deptId={deptId!} />

      <Box sx={{ px: { xs: 2, sm: 3 }, borderBottom: 1, borderColor: 'divider' }}>
        <Tabs
          value={tab}
          onChange={(_, v: Tab) => setTab(v)}
          aria-label="Department sections"
        >
          <Tab value="overview" label="Overview" />
          <Tab value="meetings" label="Meetings" />
          <Tab value="calls" label="Calls" />
        </Tabs>
      </Box>

      <Box sx={{ px: { xs: 2, sm: 3 }, py: 3 }}>
        {tab === 'overview' && <DeptOverviewPanel orgId={orgId!} deptId={deptId!} />}
        {tab === 'meetings' && (
          <ScopedMeetingsPage scope={{ type: 'department', id: deptId!, orgId: orgId! }} embedded />
        )}
        {tab === 'calls' && (
          <ScopedCallsPage scope={{ type: 'department', id: deptId!, orgId: orgId! }} embedded />
        )}
      </Box>
    </Box>
  )
}

// ─── Hero (loads dept name) ──────────────────────────────────────────────────

function DeptHero({ orgId, deptId }: { orgId: string; deptId: string }) {
  const orgName = useOrgsStore((s) => s.getOrgName(orgId))
  const { data: dept, isError } = useQuery({
    queryKey: ['dept', deptId],
    queryFn: () => organizationService.getDepartment(deptId),
  })

  if (isError) return null

  return (
    <Box sx={{ px: { xs: 2, sm: 3 }, py: 1 }}>
      <HeroHeader
        title={dept?.name ?? '\u00a0'}
        subtitle={dept?.description || (orgName ? `Department in ${orgName}` : undefined)}
      />
    </Box>
  )
}

// ─── Overview panel ──────────────────────────────────────────────────────────

function DeptOverviewPanel({ orgId, deptId }: { orgId: string; deptId: string }) {
  const queryClient = useQueryClient()
  const user = useAuthStore((s) => s.user)
  const handleStale = useStalePermissionHandler({
    invalidate: () => queryClient.invalidateQueries({ queryKey: ['dept-members', deptId] }),
  })
  const [addOpen, setAddOpen] = useState(false)
  const [userId, setUserId] = useState('')
  const [role, setRole] = useState<'head' | 'member'>('member')
  const [error, setError] = useState('')

  const { data: members, isLoading, isError } = useQuery({
    queryKey: ['dept-members', deptId],
    queryFn: () => organizationService.listDeptMembers(deptId),
  })

  // Caller's own org + dept membership for affordance gating.
  const { data: orgMembers } = useQuery({
    queryKey: ['org-members', orgId],
    queryFn: () => organizationService.listMembers(orgId),
  })
  const orgMembership = orgMembers?.find((m) => m.user_id === user?.id) ?? null
  const deptMembership = members?.find((m) => m.user_id === user?.id) ?? null

  const canAdd = canAddDeptMember(orgMembership, deptMembership, user)
  const canPromote = canPromoteDeptMember(orgMembership, user)

  const addMutation = useMutation({
    mutationFn: () => organizationService.addDeptMember(deptId, { user_id: userId, role }),
    onSuccess: () => {
      setAddOpen(false)
      setUserId('')
      setRole('member')
      setError('')
      queryClient.invalidateQueries({ queryKey: ['dept-members', deptId] })
    },
    onError: (e) => {
      if (handleStale(e)) {
        setAddOpen(false)
        return
      }
      setError(e instanceof ApiError ? e.message : 'Failed to add member')
    },
  })

  const removeMutation = useMutation({
    mutationFn: (uid: string) => organizationService.removeDeptMember(deptId, uid),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dept-members', deptId] })
    },
    onError: (e) => handleStale(e),
  })

  if (isError) return <AccessDeniedState />

  return (
    <Stack spacing={3} sx={{ maxWidth: 960, mx: 'auto' }}>
      <Stack direction="row" alignItems="center" justifyContent="space-between" spacing={2}>
        <Typography variant="h6" sx={{ fontWeight: 600 }}>
          Members
        </Typography>
        {canAdd && (
          <GradientButton
            fullWidth={false}
            size="small"
            startIcon={<PersonAddIcon />}
            onClick={() => setAddOpen(true)}
          >
            Add member
          </GradientButton>
        )}
      </Stack>

      {isLoading ? (
        <Box display="flex" justifyContent="center" py={6}>
          <CircularProgress size={28} />
        </Box>
      ) : members && members.length > 0 ? (
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>User</TableCell>
              <TableCell>Role</TableCell>
              <TableCell>Joined</TableCell>
              <TableCell align="right" />
            </TableRow>
          </TableHead>
          <TableBody>
            {members.map((m) => (
              <TableRow key={m.user_id}>
                <TableCell>{m.user_id}</TableCell>
                <TableCell>{m.role}</TableCell>
                <TableCell>
                  {new Date(m.joined_at).toLocaleDateString(undefined, {
                    year: 'numeric',
                    month: 'short',
                    day: 'numeric',
                  })}
                </TableCell>
                <TableCell align="right">
                  {(canAdd || m.user_id === user?.id) && (
                    <IconButton
                      size="small"
                      onClick={() => removeMutation.mutate(m.user_id)}
                      aria-label={m.user_id === user?.id ? 'Leave department' : 'Remove member'}
                    >
                      <DeleteIcon fontSize="small" />
                    </IconButton>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      ) : (
        <Typography variant="body2" color="text.secondary">
          No members yet.
        </Typography>
      )}

      <Dialog open={addOpen} onClose={() => setAddOpen(false)} fullWidth maxWidth="xs">
        <DialogTitle>Add department member</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            <TextField
              label="User ID"
              value={userId}
              onChange={(e) => setUserId(e.target.value)}
              fullWidth
              autoFocus
              helperText="UUID of an existing organization member."
            />
            <Select<'head' | 'member'>
              value={role}
              onChange={(e) => setRole(e.target.value as 'head' | 'member')}
              fullWidth
            >
              <MenuItem value="member">Member</MenuItem>
              {/* Promotion to head is reserved for org admins / platform admins. */}
              {canPromote && <MenuItem value="head">Head</MenuItem>}
            </Select>
            {error && <Alert severity="error">{error}</Alert>}
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setAddOpen(false)}>Cancel</Button>
          <GradientButton
            fullWidth={false}
            onClick={() => addMutation.mutate()}
            disabled={!isUuid(userId) || addMutation.isPending}
          >
            Add
          </GradientButton>
        </DialogActions>
      </Dialog>

      {/* Suppress unused-var warning for orgId; reserved for future controls. */}
      <input type="hidden" data-testid="dept-overview-org-id" value={orgId} />
    </Stack>
  )
}
