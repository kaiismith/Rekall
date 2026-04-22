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
    onSuccess: () => navigate(ROUTES.ORGANIZATIONS, { replace: true }),
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
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['org-departments', id] }),
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
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 3 }}>
        <Box>
          <Typography variant="h5" fontWeight={700}>{org.name}</Typography>
          <Typography variant="body2" color="text.secondary">/{org.slug}</Typography>
        </Box>
        {canManage && (
          <Button startIcon={<PersonAddIcon />} variant="outlined" onClick={() => {
            setInviteSuccess(false)
            setInviteError('')
            setInviteOpen(true)
          }}>
            Invite member
          </Button>
        )}
      </Box>

      <Divider sx={{ mb: 3 }} />

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
                <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.8rem' }}>{m.user_id}</TableCell>
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
              if (confirm(`Delete department "${dept.name}"? This cannot be undone.`)) {
                deleteDeptMutation.mutate(dept.id)
              }
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
        <Box sx={{ mt: 6, pt: 3, borderTop: '1px solid', borderColor: 'divider' }}>
          <Typography variant="h6" color="error" sx={{ mb: 1 }}>Danger zone</Typography>
          <Button
            variant="outlined"
            color="error"
            onClick={() => {
              if (confirm(`Delete "${org.name}"? This cannot be undone.`)) {
                deleteMutation.mutate()
              }
            }}
            disabled={deleteMutation.isPending}
          >
            Delete organization
          </Button>
        </Box>
      )}

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
                      <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.8rem' }}>{m.user_id}</TableCell>
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
