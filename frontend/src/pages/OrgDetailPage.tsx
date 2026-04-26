import { useState } from 'react'
import { useParams, useNavigate, useSearchParams } from 'react-router-dom'
import {
  Alert,
  Box,
  Button,
  Card,
  CardActionArea,
  CardContent,
  Chip,
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
import AddIcon from '@mui/icons-material/Add'
import DeleteIcon from '@mui/icons-material/Delete'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { organizationService } from '@/services/organizationService'
import { useAuthStore } from '@/store/authStore'
import { ApiError } from '@/services/api'
import { ROUTES, buildScopedRoute } from '@/constants'
import {
  AccessDeniedState,
  ConfirmDeleteDialog,
  GradientButton,
  HeroHeader,
  PageHeader,
  ScopeBreadcrumb,
} from '@/components/common/ui'
import { ScopedMeetingsPage } from './ScopedMeetingsPage'
import { ScopedCallsPage } from './ScopedCallsPage'
import { tokens } from '@/theme'
import type { Department, OrgMember } from '@/types/organization'
import type { User } from '@/types/auth'
import { canManageDept, canManageOrg } from '@/utils/permissions'
import { useStalePermissionHandler } from '@/hooks/useStalePermissionHandler'
import { useOrgsStore } from '@/store/orgsStore'
import { useDeptsStore } from '@/store/deptsStore'

const TABS = ['overview', 'departments', 'meetings', 'calls'] as const
type TabKey = (typeof TABS)[number]

function normalizeTab(value: string | null): TabKey {
  return TABS.includes(value as TabKey) ? (value as TabKey) : 'overview'
}

export function OrgDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [searchParams, setSearchParams] = useSearchParams()
  const tab = normalizeTab(searchParams.get('tab'))
  const { user } = useAuthStore()

  const { data: org, isLoading, isError, error } = useQuery({
    queryKey: ['org', id],
    queryFn: () => organizationService.get(id!),
    enabled: !!id,
  })

  const { data: members } = useQuery({
    queryKey: ['org-members', id],
    queryFn: () => organizationService.listMembers(id!),
    enabled: !!id,
  })

  const setTab = (next: TabKey) => {
    setSearchParams(
      (prev) => {
        const out = new URLSearchParams(prev)
        out.set('tab', next)
        return out
      },
      { replace: true },
    )
  }

  if (isLoading) {
    return (
      <Box display="flex" justifyContent="center" py={8}>
        <CircularProgress />
      </Box>
    )
  }

  // 403 / 404 from the org fetch — non-member or stale link.
  if (isError) {
    const status =
      error instanceof ApiError ? error.status : (error as { response?: { status?: number } })?.response?.status
    if (status === 403 || status === 404) return <AccessDeniedState />
    return <Alert severity="error">{(error as ApiError).message}</Alert>
  }
  if (!org) return null

  const currentMember = members?.find((m) => m.user_id === user?.id)

  return (
    <Box>
      <ScopeBreadcrumb />

      <Box sx={{ px: { xs: 2, sm: 3 }, pt: 2, pb: 1 }}>
        <HeroHeader title={org.name} subtitle={`/${org.slug}`} />
      </Box>

      <Box sx={{ px: { xs: 2, sm: 3 }, borderBottom: 1, borderColor: 'divider' }}>
        <Tabs value={tab} onChange={(_, v: TabKey) => setTab(v)} aria-label="Organization sections">
          <Tab value="overview" label="Overview" />
          <Tab value="departments" label="Departments" />
          <Tab value="meetings" label="Meetings" />
          <Tab value="calls" label="Calls" />
        </Tabs>
      </Box>

      <Box sx={{ px: { xs: 2, sm: 3 }, py: 3 }}>
        {tab === 'overview' && (
          <OrgOverviewPanel orgId={id!} org={org} currentMember={currentMember} user={user} />
        )}
        {tab === 'departments' && (
          <OrgDepartmentsPanel orgId={id!} currentMember={currentMember} user={user} />
        )}
        {tab === 'meetings' && (
          <ScopedMeetingsPage scope={{ type: 'organization', id: id! }} embedded />
        )}
        {tab === 'calls' && <ScopedCallsPage scope={{ type: 'organization', id: id! }} embedded />}
      </Box>
    </Box>
  )
}

// ── Overview panel: members + invite + danger zone ──────────────────────────

interface OverviewProps {
  orgId: string
  org: { id: string; name: string; slug: string }
  currentMember: OrgMember | undefined
  user: User | null
}

function OrgOverviewPanel({ orgId, org, currentMember, user }: OverviewProps) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const invalidateOrgs = useOrgsStore((s) => s.invalidate)
  const handleStale = useStalePermissionHandler({
    invalidate: () => queryClient.invalidateQueries({ queryKey: ['org-members', orgId] }),
  })

  const canManage = canManageOrg(currentMember ?? null, user)
  const isOwner = currentMember?.role === 'owner'

  const { data: members, isLoading: membersLoading } = useQuery({
    queryKey: ['org-members', orgId],
    queryFn: () => organizationService.listMembers(orgId),
  })

  const [inviteOpen, setInviteOpen] = useState(false)
  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteRole, setInviteRole] = useState<'member' | 'admin'>('member')
  const [inviteError, setInviteError] = useState('')
  const [inviteSuccess, setInviteSuccess] = useState(false)
  const [deleteOrgOpen, setDeleteOrgOpen] = useState(false)
  const [deleteOrgError, setDeleteOrgError] = useState<string | null>(null)

  const inviteMutation = useMutation({
    mutationFn: () => organizationService.inviteUser(orgId, { email: inviteEmail, role: inviteRole }),
    onSuccess: () => {
      setInviteSuccess(true)
      setInviteEmail('')
    },
    onError: (err) => {
      if (handleStale(err)) return
      setInviteError(err instanceof ApiError ? err.message : 'Failed to send invitation')
    },
  })

  const removeMutation = useMutation({
    mutationFn: (userId: string) => organizationService.removeMember(orgId, userId),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['org-members', orgId] }),
    onError: (err) => handleStale(err),
  })

  const deleteMutation = useMutation({
    mutationFn: () => organizationService.delete(orgId),
    onSuccess: () => {
      setDeleteOrgOpen(false)
      invalidateOrgs()
      navigate(ROUTES.ORGANIZATIONS, { replace: true })
    },
    onError: (err) => {
      if (handleStale(err)) {
        setDeleteOrgOpen(false)
        return
      }
      setDeleteOrgError(err instanceof ApiError ? err.message : 'Failed to delete organization')
    },
  })

  return (
    <Stack spacing={4}>
      <Box>
        <PageHeader
          title="Members"
          actions={
            canManage ? (
              <GradientButton
                size="small"
                fullWidth={false}
                startIcon={<PersonAddIcon />}
                onClick={() => {
                  setInviteSuccess(false)
                  setInviteError('')
                  setInviteOpen(true)
                }}
              >
                Invite member
              </GradientButton>
            ) : undefined
          }
        />

        {membersLoading ? (
          <CircularProgress size={24} />
        ) : (
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>User ID</TableCell>
                <TableCell>Role</TableCell>
                <TableCell>Joined</TableCell>
                {canManage && <TableCell />}
              </TableRow>
            </TableHead>
            <TableBody>
              {members?.map((m) => (
                <TableRow key={m.user_id}>
                  <TableCell sx={{ fontFamily: tokens.fonts.mono, fontSize: '0.8rem' }}>
                    {m.user_id}
                  </TableCell>
                  <TableCell sx={{ textTransform: 'capitalize' }}>{m.role}</TableCell>
                  <TableCell>{new Date(m.joined_at).toLocaleDateString()}</TableCell>
                  {canManage && (
                    <TableCell align="right">
                      {m.role !== 'owner' && m.user_id !== user?.id && (
                        <IconButton
                          size="small"
                          color="error"
                          onClick={() => removeMutation.mutate(m.user_id)}
                          disabled={removeMutation.isPending}
                        >
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      )}
                    </TableCell>
                  )}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Box>

      {isOwner && (
        <Box
          sx={{
            p: 3,
            borderRadius: '12px',
            border: '1px solid rgba(239,68,68,0.25)',
            bgcolor: 'rgba(239,68,68,0.04)',
          }}
        >
          <Typography
            variant="overline"
            sx={{ color: '#fca5a5', fontWeight: 700, letterSpacing: '0.12em', display: 'block', mb: 0.5 }}
          >
            Danger zone
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            Deleting this organization will permanently remove all members, departments, and related
            records.
          </Typography>
          <Button
            variant="outlined"
            color="error"
            onClick={() => {
              setDeleteOrgError(null)
              setDeleteOrgOpen(true)
            }}
            disabled={deleteMutation.isPending}
          >
            Delete organization
          </Button>
        </Box>
      )}

      <ConfirmDeleteDialog
        open={deleteOrgOpen}
        onClose={() => setDeleteOrgOpen(false)}
        onConfirm={() => deleteMutation.mutate()}
        title="Delete organization"
        entityName={org.name}
        confirmationValue={org.slug}
        confirmationLabel="slug"
        description={`You're about to permanently delete "${org.name}". Other organizations may share this name, so we use the slug to make sure you're deleting the right one.`}
        consequences={[
          'Remove all members and their access',
          'Delete every department and its memberships',
          'Cancel pending invitations to this organization',
        ]}
        confirmLabel="Delete organization"
        loading={deleteMutation.isPending}
        error={deleteOrgError}
      />

      <Dialog open={inviteOpen} onClose={() => setInviteOpen(false)} maxWidth="xs" fullWidth>
        <DialogTitle>Invite a member</DialogTitle>
        <DialogContent
          sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: '16px !important' }}
        >
          {inviteError && <Alert severity="error">{inviteError}</Alert>}
          {inviteSuccess && (
            <Alert severity="success">Invitation sent to {inviteEmail || 'the user'}.</Alert>
          )}
          <TextField
            label="Email address"
            type="email"
            value={inviteEmail}
            onChange={(e) => setInviteEmail(e.target.value)}
            required
            fullWidth
            autoFocus
          />
          <Select
            value={inviteRole}
            onChange={(e) => setInviteRole(e.target.value as 'member' | 'admin')}
            size="small"
          >
            <MenuItem value="member">Member</MenuItem>
            <MenuItem value="admin">Admin</MenuItem>
          </Select>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setInviteOpen(false)}>Close</Button>
          <Button
            onClick={() => {
              setInviteError('')
              setInviteSuccess(false)
              inviteMutation.mutate()
            }}
            variant="contained"
            disabled={!inviteEmail.trim() || inviteMutation.isPending}
          >
            {inviteMutation.isPending ? 'Sending…' : 'Send invitation'}
          </Button>
        </DialogActions>
      </Dialog>
    </Stack>
  )
}

// ── Departments panel: list + create; cards link to DeptDetailPage ──────────

interface DeptPanelProps {
  orgId: string
  currentMember: OrgMember | undefined
  user: User | null
}

function OrgDepartmentsPanel({ orgId, currentMember, user }: DeptPanelProps) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const invalidateDepts = useDeptsStore((s) => s.invalidate)
  const handleStale = useStalePermissionHandler({
    invalidate: () => queryClient.invalidateQueries({ queryKey: ['org-departments', orgId] }),
  })
  const canManage = canManageDept(currentMember ?? null, user)

  const { data: departments } = useQuery({
    queryKey: ['org-departments', orgId],
    queryFn: () => organizationService.listDepartments(orgId),
  })

  const [createOpen, setCreateOpen] = useState(false)
  const [deptName, setDeptName] = useState('')
  const [deptDescription, setDeptDescription] = useState('')
  const [createError, setCreateError] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<Department | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  const createMutation = useMutation({
    mutationFn: () =>
      organizationService.createDepartment(orgId, { name: deptName, description: deptDescription }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['org-departments', orgId] })
      invalidateDepts(orgId)
      setCreateOpen(false)
      setDeptName('')
      setDeptDescription('')
    },
    onError: (err) => {
      if (handleStale(err)) {
        setCreateOpen(false)
        return
      }
      setCreateError(err instanceof ApiError ? err.message : 'Failed to create department')
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (deptId: string) => organizationService.deleteDepartment(deptId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['org-departments', orgId] })
      invalidateDepts(orgId)
      setDeleteTarget(null)
    },
    onError: (err) => {
      if (handleStale(err)) {
        setDeleteTarget(null)
        return
      }
      setDeleteError(err instanceof ApiError ? err.message : 'Failed to delete department')
    },
  })

  return (
    <Stack spacing={3}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography variant="h6" sx={{ fontWeight: 600 }}>
          Departments
        </Typography>
        {canManage && (
          <Button
            startIcon={<AddIcon />}
            variant="outlined"
            size="small"
            onClick={() => {
              setCreateError('')
              setCreateOpen(true)
            }}
          >
            New department
          </Button>
        )}
      </Box>

      {departments && departments.length === 0 && (
        <Typography color="text.secondary" variant="body2">
          No departments yet.{canManage ? ' Create one to organise your team.' : ''}
        </Typography>
      )}

      <Box
        sx={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))',
          gap: 2,
        }}
      >
        {departments?.map((dept: Department) => (
          <Card key={dept.id} variant="outlined">
            <CardActionArea onClick={() => navigate(buildScopedRoute.dept(orgId, dept.id))}>
              <CardContent>
                <Stack
                  direction="row"
                  justifyContent="space-between"
                  alignItems="flex-start"
                  spacing={1}
                >
                  <Box sx={{ minWidth: 0 }}>
                    <Typography variant="subtitle1" fontWeight={600} noWrap>
                      {dept.name}
                    </Typography>
                    {dept.description && (
                      <Typography variant="body2" color="text.secondary" noWrap>
                        {dept.description}
                      </Typography>
                    )}
                  </Box>
                  {canManage && (
                    <IconButton
                      size="small"
                      color="error"
                      aria-label="Delete department"
                      onClick={(e) => {
                        e.stopPropagation()
                        e.preventDefault()
                        setDeleteError(null)
                        setDeleteTarget(dept)
                      }}
                    >
                      <DeleteIcon fontSize="small" />
                    </IconButton>
                  )}
                </Stack>
                <Chip
                  label="Open"
                  size="small"
                  sx={{ mt: 1.5, fontSize: '0.7rem', height: 22 }}
                />
              </CardContent>
            </CardActionArea>
          </Card>
        ))}
      </Box>

      <ConfirmDeleteDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
        title="Delete department"
        entityName={deleteTarget?.name ?? ''}
        confirmationValue={deleteTarget?.id ?? ''}
        confirmationLabel="department ID"
        description={`You're about to permanently delete the department "${deleteTarget?.name ?? ''}". Type its ID to confirm — this is unique even if another department shares the same name.`}
        consequences={['Remove all department memberships', 'This action cannot be undone']}
        confirmLabel="Delete department"
        loading={deleteMutation.isPending}
        error={deleteError}
      />

      <Dialog open={createOpen} onClose={() => setCreateOpen(false)} maxWidth="xs" fullWidth>
        <DialogTitle>New department</DialogTitle>
        <DialogContent
          sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: '16px !important' }}
        >
          {createError && <Alert severity="error">{createError}</Alert>}
          <TextField
            label="Department name"
            value={deptName}
            onChange={(e) => setDeptName(e.target.value)}
            required
            fullWidth
            autoFocus
          />
          <TextField
            label="Description (optional)"
            value={deptDescription}
            onChange={(e) => setDeptDescription(e.target.value)}
            fullWidth
            multiline
            rows={2}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateOpen(false)}>Cancel</Button>
          <Button
            onClick={() => {
              setCreateError('')
              createMutation.mutate()
            }}
            variant="contained"
            disabled={!deptName.trim() || createMutation.isPending}
          >
            {createMutation.isPending ? 'Creating…' : 'Create'}
          </Button>
        </DialogActions>
      </Dialog>
    </Stack>
  )
}
