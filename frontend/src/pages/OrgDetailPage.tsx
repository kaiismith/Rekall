import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Collapse,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  IconButton,
  MenuItem,
  Select,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from '@mui/material'
import PersonAddIcon from '@mui/icons-material/PersonAdd'
import AddIcon from '@mui/icons-material/Add'
import DeleteIcon from '@mui/icons-material/Delete'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'
import ExpandLessIcon from '@mui/icons-material/ExpandLess'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { organizationService } from '@/services/organizationService'
import { useAuthStore } from '@/store/authStore'
import { ApiError } from '@/services/api'
import { ROUTES } from '@/constants'
import { ConfirmDeleteDialog, GradientButton, PageHeader } from '@/components/common/ui'
import { tokens } from '@/theme'
import type { Department } from '@/types/organization'

export function OrgDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { user } = useAuthStore()

  // Invite member state
  const [inviteOpen, setInviteOpen] = useState(false)
  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteRole, setInviteRole] = useState<'member' | 'admin'>('member')
  const [inviteError, setInviteError] = useState('')
  const [inviteSuccess, setInviteSuccess] = useState(false)

  // Create department state
  const [createDeptOpen, setCreateDeptOpen] = useState(false)
  const [deptName, setDeptName] = useState('')
  const [deptDescription, setDeptDescription] = useState('')
  const [createDeptError, setCreateDeptError] = useState('')

  // Add dept member state
  const [addMemberDeptId, setAddMemberDeptId] = useState<string | null>(null)
  const [addMemberUserId, setAddMemberUserId] = useState('')
  const [addMemberRole, setAddMemberRole] = useState<'member' | 'head'>('member')
  const [addMemberError, setAddMemberError] = useState('')

  // Expanded department cards
  const [expandedDepts, setExpandedDepts] = useState<Record<string, boolean>>({})

  // Delete confirmation dialogs
  const [deleteOrgOpen, setDeleteOrgOpen] = useState(false)
  const [deleteOrgError, setDeleteOrgError] = useState<string | null>(null)
  const [deleteDeptTarget, setDeleteDeptTarget] = useState<Department | null>(null)
  const [deleteDeptError, setDeleteDeptError] = useState<string | null>(null)

  const { data: org, isLoading: orgLoading, error: orgError } = useQuery({
    queryKey: ['org', id],
    queryFn: () => organizationService.get(id!),
    enabled: !!id,
  })

  const { data: members, isLoading: membersLoading } = useQuery({
    queryKey: ['org-members', id],
    queryFn: () => organizationService.listMembers(id!),
    enabled: !!id,
  })

  const { data: departments } = useQuery({
    queryKey: ['org-departments', id],
    queryFn: () => organizationService.listDepartments(id!),
    enabled: !!id,
  })

  const inviteMutation = useMutation({
    mutationFn: () => organizationService.inviteUser(id!, { email: inviteEmail, role: inviteRole }),
    onSuccess: () => {
      setInviteSuccess(true)
      setInviteEmail('')
    },
    onError: (err) => {
      setInviteError(err instanceof ApiError ? err.message : 'Failed to send invitation')
    },
  })

  const removeMutation = useMutation({
    mutationFn: (userId: string) => organizationService.removeMember(id!, userId),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['org-members', id] }),
  })

  const deleteMutation = useMutation({
    mutationFn: () => organizationService.delete(id!),
    onSuccess: () => {
      setDeleteOrgOpen(false)
      navigate(ROUTES.ORGANIZATIONS, { replace: true })
    },
    onError: (err) => {
      setDeleteOrgError(err instanceof ApiError ? err.message : 'Failed to delete organization')
    },
  })

  const createDeptMutation = useMutation({
    mutationFn: () => organizationService.createDepartment(id!, { name: deptName, description: deptDescription }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['org-departments', id] })
      setCreateDeptOpen(false)
      setDeptName('')
      setDeptDescription('')
    },
    onError: (err) => {
      setCreateDeptError(err instanceof ApiError ? err.message : 'Failed to create department')
    },
  })

  const deleteDeptMutation = useMutation({
    mutationFn: (deptId: string) => organizationService.deleteDepartment(deptId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['org-departments', id] })
      setDeleteDeptTarget(null)
    },
    onError: (err) => {
      setDeleteDeptError(err instanceof ApiError ? err.message : 'Failed to delete department')
    },
  })

  const addDeptMemberMutation = useMutation({
    mutationFn: () => organizationService.addDeptMember(addMemberDeptId!, { user_id: addMemberUserId, role: addMemberRole }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['dept-members', addMemberDeptId] })
      setAddMemberDeptId(null)
      setAddMemberUserId('')
    },
    onError: (err) => {
      setAddMemberError(err instanceof ApiError ? err.message : 'Failed to add member')
    },
  })

  if (orgLoading) return <CircularProgress />
  if (orgError) return <Alert severity="error">{(orgError as ApiError).message}</Alert>
  if (!org) return null

  const currentMember = members?.find((m) => m.user_id === user?.id)
  const isOwner = currentMember?.role === 'owner'
  const canManage = currentMember?.role === 'owner' || currentMember?.role === 'admin'

  const toggleDept = (deptId: string) =>
    setExpandedDepts((prev) => ({ ...prev, [deptId]: !prev[deptId] }))

  return (
    <Box>
      <PageHeader
        eyebrow="Organization"
        title={org.name}
        subtitle={`/${org.slug}`}
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

      {/* ── Members ─────────────────────────────────────────────────── */}
      <Typography variant="h6" sx={{ mb: 2 }}>Members</Typography>

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
                <TableCell sx={{ fontFamily: tokens.fonts.mono, fontSize: '0.8rem' }}>{m.user_id}</TableCell>
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

      <Divider sx={{ my: 4 }} />

      {/* ── Departments ──────────────────────────────────────────────── */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Typography variant="h6">Departments</Typography>
        {canManage && (
          <Button
            startIcon={<AddIcon />}
            variant="outlined"
            size="small"
            onClick={() => { setCreateDeptError(''); setCreateDeptOpen(true) }}
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

      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
        {departments?.map((dept: Department) => (
          <DepartmentCard
            key={dept.id}
            dept={dept}
            canManage={canManage}
            expanded={!!expandedDepts[dept.id]}
            onToggle={() => toggleDept(dept.id)}
            onDelete={() => {
              setDeleteDeptError(null)
              setDeleteDeptTarget(dept)
            }}
            onAddMember={() => {
              setAddMemberError('')
              setAddMemberUserId('')
              setAddMemberRole('member')
              setAddMemberDeptId(dept.id)
            }}
          />
        ))}
      </Box>

      {/* Danger zone */}
      {isOwner && (
        <Box
          sx={{
            mt: 6,
            p: 3,
            borderRadius: '12px',
            border: '1px solid rgba(239,68,68,0.25)',
            bgcolor: 'rgba(239,68,68,0.04)',
          }}
        >
          <Typography
            variant="overline"
            sx={{
              color: '#fca5a5',
              fontWeight: 700,
              letterSpacing: '0.12em',
              display: 'block',
              mb: 0.5,
            }}
          >
            Danger zone
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            Deleting this organization will permanently remove all members, departments,
            and related records.
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

      {/* Typed-confirmation dialogs */}
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

      <ConfirmDeleteDialog
        open={!!deleteDeptTarget}
        onClose={() => setDeleteDeptTarget(null)}
        onConfirm={() => deleteDeptTarget && deleteDeptMutation.mutate(deleteDeptTarget.id)}
        title="Delete department"
        entityName={deleteDeptTarget?.name ?? ''}
        confirmationValue={deleteDeptTarget?.id ?? ''}
        confirmationLabel="department ID"
        description={`You're about to permanently delete the department "${deleteDeptTarget?.name ?? ''}". Type its ID to confirm — this is unique even if another department shares the same name.`}
        consequences={[
          'Remove all department memberships',
          'This action cannot be undone',
        ]}
        confirmLabel="Delete department"
        loading={deleteDeptMutation.isPending}
        error={deleteDeptError}
      />

      {/* ── Invite dialog ────────────────────────────────────────────── */}
      <Dialog open={inviteOpen} onClose={() => setInviteOpen(false)} maxWidth="xs" fullWidth>
        <DialogTitle>Invite a member</DialogTitle>
        <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: '16px !important' }}>
          {inviteError && <Alert severity="error">{inviteError}</Alert>}
          {inviteSuccess && <Alert severity="success">Invitation sent to {inviteEmail || 'the user'}.</Alert>}
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

      {/* ── Create department dialog ─────────────────────────────────── */}
      <Dialog open={createDeptOpen} onClose={() => setCreateDeptOpen(false)} maxWidth="xs" fullWidth>
        <DialogTitle>New department</DialogTitle>
        <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: '16px !important' }}>
          {createDeptError && <Alert severity="error">{createDeptError}</Alert>}
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
          <Button onClick={() => setCreateDeptOpen(false)}>Cancel</Button>
          <Button
            onClick={() => { setCreateDeptError(''); createDeptMutation.mutate() }}
            variant="contained"
            disabled={!deptName.trim() || createDeptMutation.isPending}
          >
            {createDeptMutation.isPending ? 'Creating…' : 'Create'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* ── Add dept member dialog ───────────────────────────────────── */}
      <Dialog open={!!addMemberDeptId} onClose={() => setAddMemberDeptId(null)} maxWidth="xs" fullWidth>
        <DialogTitle>Add member to department</DialogTitle>
        <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: '16px !important' }}>
          {addMemberError && <Alert severity="error">{addMemberError}</Alert>}
          <TextField
            label="User ID"
            value={addMemberUserId}
            onChange={(e) => setAddMemberUserId(e.target.value)}
            required
            fullWidth
            autoFocus
            helperText="Must be an existing org member"
          />
          <Select
            value={addMemberRole}
            onChange={(e) => setAddMemberRole(e.target.value as 'member' | 'head')}
            size="small"
          >
            <MenuItem value="member">Member</MenuItem>
            {canManage && <MenuItem value="head">Head (team leader)</MenuItem>}
          </Select>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setAddMemberDeptId(null)}>Cancel</Button>
          <Button
            onClick={() => { setAddMemberError(''); addDeptMemberMutation.mutate() }}
            variant="contained"
            disabled={!addMemberUserId.trim() || addDeptMemberMutation.isPending}
          >
            {addDeptMemberMutation.isPending ? 'Adding…' : 'Add'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}

// ── DepartmentCard ────────────────────────────────────────────────────────────

interface DepartmentCardProps {
  dept: Department
  canManage: boolean
  expanded: boolean
  onToggle: () => void
  onDelete: () => void
  onAddMember: () => void
}

function DepartmentCard({ dept, canManage, expanded, onToggle, onDelete, onAddMember }: DepartmentCardProps) {
  const { data: members, isLoading } = useQuery({
    queryKey: ['dept-members', dept.id],
    queryFn: () => organizationService.listDeptMembers(dept.id),
    enabled: expanded,
  })

  const queryClient = useQueryClient()

  const removeMutation = useMutation({
    mutationFn: (userId: string) => organizationService.removeDeptMember(dept.id, userId),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['dept-members', dept.id] }),
  })

  const heads = members?.filter((m) => m.role === 'head') ?? []

  return (
    <Card variant="outlined">
      <CardContent sx={{ pb: '8px !important' }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
          <Box>
            <Typography variant="subtitle1" fontWeight={600}>{dept.name}</Typography>
            {dept.description && (
              <Typography variant="body2" color="text.secondary">{dept.description}</Typography>
            )}
            {heads.length > 0 && (
              <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap', mt: 0.5 }}>
                {heads.map((h) => (
                  <Chip key={h.user_id} label={`Head: ${h.user_id.slice(0, 8)}…`} size="small" color="primary" variant="outlined" />
                ))}
              </Box>
            )}
          </Box>
          <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
            {canManage && (
              <>
                <Button size="small" startIcon={<PersonAddIcon />} onClick={onAddMember}>
                  Add
                </Button>
                <IconButton size="small" color="error" onClick={onDelete}>
                  <DeleteIcon fontSize="small" />
                </IconButton>
              </>
            )}
            <IconButton size="small" onClick={onToggle}>
              {expanded ? <ExpandLessIcon fontSize="small" /> : <ExpandMoreIcon fontSize="small" />}
            </IconButton>
          </Box>
        </Box>

        <Collapse in={expanded}>
          <Box sx={{ mt: 2 }}>
            {isLoading ? (
              <CircularProgress size={20} />
            ) : members && members.length > 0 ? (
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
                  {members.map((m) => (
                    <TableRow key={m.user_id}>
                      <TableCell sx={{ fontFamily: tokens.fonts.mono, fontSize: '0.8rem' }}>{m.user_id}</TableCell>
                      <TableCell>
                        <Chip
                          label={m.role}
                          size="small"
                          color={m.role === 'head' ? 'primary' : 'default'}
                          sx={{ textTransform: 'capitalize' }}
                        />
                      </TableCell>
                      <TableCell>{new Date(m.joined_at).toLocaleDateString()}</TableCell>
                      {canManage && (
                        <TableCell align="right">
                          <IconButton
                            size="small"
                            color="error"
                            onClick={() => removeMutation.mutate(m.user_id)}
                            disabled={removeMutation.isPending}
                          >
                            <DeleteIcon fontSize="small" />
                          </IconButton>
                        </TableCell>
                      )}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : (
              <Typography variant="body2" color="text.secondary">No members in this department yet.</Typography>
            )}
          </Box>
        </Collapse>
      </CardContent>
    </Card>
  )
}
