import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Alert,
  Box,
  Button,
  Card,
  CardActionArea,
  CardContent,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import AddIcon from '@mui/icons-material/Add'
import ApartmentOutlinedIcon from '@mui/icons-material/ApartmentOutlined'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { organizationService } from '@/services/organizationService'
import { ApiError } from '@/services/api'
import { ROUTES } from '@/constants'
import { EmptyState, GradientButton, PageHeader } from '@/components/common/ui'
import { tokens } from '@/theme'
import { useAuthStore } from '@/store/authStore'
import { useOrgsStore } from '@/store/orgsStore'
import { canCreateOrg } from '@/utils/permissions'
import { useStalePermissionHandler } from '@/hooks/useStalePermissionHandler'

export function OrganizationsPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const user = useAuthStore((s) => s.user)
  const invalidateOrgs = useOrgsStore((s) => s.invalidate)
  const adminCanCreate = canCreateOrg(user)
  const handleStale = useStalePermissionHandler({ invalidate: invalidateOrgs })

  const [createOpen, setCreateOpen] = useState(false)
  const [name, setName] = useState('')
  const [ownerEmail, setOwnerEmail] = useState('')
  const [createError, setCreateError] = useState('')

  const { data: orgs, isLoading, error } = useQuery({
    queryKey: ['organizations'],
    queryFn: organizationService.list,
  })

  const createMutation = useMutation({
    mutationFn: (input: { name: string; owner_email?: string }) =>
      organizationService.create(input),
    onSuccess: (org) => {
      // Invalidate both the React Query cache (used by this page) and the
      // global orgs store (used by OrgSwitcher / ScopeBadge / ScopePicker).
      void queryClient.invalidateQueries({ queryKey: ['organizations'] })
      invalidateOrgs()
      setCreateOpen(false)
      setName('')
      setOwnerEmail('')
      navigate(ROUTES.ORG_DETAIL.replace(':id', org.id))
    },
    onError: (err) => {
      if (handleStale(err)) {
        setCreateOpen(false)
        return
      }
      setCreateError(err instanceof ApiError ? err.message : 'Failed to create organization')
    },
  })

  const handleCreate = () => {
    setCreateError('')
    createMutation.mutate(
      ownerEmail ? { name, owner_email: ownerEmail } : { name },
    )
  }

  const openDialog = () => {
    setCreateError('')
    setCreateOpen(true)
  }

  const showEmpty = !isLoading && !error && orgs && orgs.length === 0

  return (
    <Box>
      <PageHeader
        title="Organizations"
        subtitle="Workspaces you belong to. Manage members, departments, and invitations."
        actions={
          adminCanCreate ? (
            <GradientButton
              size="small"
              fullWidth={false}
              startIcon={<AddIcon />}
              onClick={openDialog}
            >
              New organization
            </GradientButton>
          ) : null
        }
      />

      {isLoading && (
        <Box display="flex" justifyContent="center" py={8}>
          <CircularProgress />
        </Box>
      )}

      {error && <Alert severity="error" sx={{ mb: 3 }}>{(error as ApiError).message}</Alert>}

      {showEmpty && (
        <EmptyState
          icon={<ApartmentOutlinedIcon />}
          title="No organizations yet"
          description={
            adminCanCreate
              ? 'Create an organization to invite teammates, manage departments, and scope private meetings.'
              : 'Contact your administrator to be added to an organization.'
          }
          action={
            adminCanCreate ? (
              <GradientButton
                fullWidth={false}
                startIcon={<AddIcon />}
                onClick={openDialog}
              >
                Create an organization
              </GradientButton>
            ) : null
          }
        />
      )}

      {orgs && orgs.length > 0 && (
        <Box
          sx={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))',
            gap: 2,
          }}
        >
          {orgs.map((org) => (
            <Card
              key={org.id}
              sx={{
                transition: 'transform 160ms ease, border-color 160ms ease',
                '&:hover': {
                  transform: 'translateY(-1px)',
                  borderColor: 'rgba(167,139,250,0.35)',
                },
              }}
            >
              <CardActionArea onClick={() => navigate(ROUTES.ORG_DETAIL.replace(':id', org.id))}>
                <CardContent sx={{ p: 2.5 }}>
                  <Stack direction="row" spacing={2} alignItems="center">
                    <Box
                      sx={{
                        width: 44,
                        height: 44,
                        borderRadius: '10px',
                        bgcolor: 'rgba(129,140,248,0.12)',
                        color: '#a78bfa',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        fontWeight: 700,
                        fontSize: '1.125rem',
                        flexShrink: 0,
                      }}
                    >
                      {org.name.slice(0, 1).toUpperCase()}
                    </Box>
                    <Box sx={{ minWidth: 0 }}>
                      <Typography
                        variant="subtitle1"
                        fontWeight={600}
                        noWrap
                        title={org.name}
                      >
                        {org.name}
                      </Typography>
                      <Typography
                        variant="caption"
                        color="text.secondary"
                        sx={{ fontFamily: tokens.fonts.mono }}
                      >
                        /{org.slug}
                      </Typography>
                    </Box>
                  </Stack>
                </CardContent>
              </CardActionArea>
            </Card>
          ))}
        </Box>
      )}

      <Dialog open={createOpen} onClose={() => setCreateOpen(false)} maxWidth="xs" fullWidth>
        <DialogTitle sx={{ fontWeight: 600 }}>Create organization</DialogTitle>
        <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: '16px !important' }}>
          {createError && <Alert severity="error">{createError}</Alert>}
          <TextField
            label="Organization name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            fullWidth
            autoFocus
            onKeyDown={(e) => e.key === 'Enter' && handleCreate()}
          />
          <TextField
            label="Owner email (optional)"
            value={ownerEmail}
            onChange={(e) => setOwnerEmail(e.target.value)}
            type="email"
            fullWidth
            helperText="Leave blank to keep yourself as owner. Otherwise, the named user becomes the org's owner."
          />
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setCreateOpen(false)}>Cancel</Button>
          <GradientButton
            onClick={handleCreate}
            fullWidth={false}
            disabled={!name.trim() || createMutation.isPending}
          >
            {createMutation.isPending ? 'Creating…' : 'Create'}
          </GradientButton>
        </DialogActions>
      </Dialog>
    </Box>
  )
}
