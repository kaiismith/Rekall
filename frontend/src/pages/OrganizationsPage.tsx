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
  TextField,
  Typography,
} from '@mui/material'
import AddIcon from '@mui/icons-material/Add'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { organizationService } from '@/services/organizationService'
import { ApiError } from '@/services/api'
import { ROUTES } from '@/constants'

export function OrganizationsPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const [createOpen, setCreateOpen] = useState(false)
  const [name, setName] = useState('')
  const [createError, setCreateError] = useState('')

  const { data: orgs, isLoading, error } = useQuery({
    queryKey: ['organizations'],
    queryFn: organizationService.list,
  })

  const createMutation = useMutation({
    mutationFn: (orgName: string) => organizationService.create({ name: orgName }),
    onSuccess: (org) => {
      void queryClient.invalidateQueries({ queryKey: ['organizations'] })
      setCreateOpen(false)
      setName('')
      navigate(ROUTES.ORG_DETAIL.replace(':id', org.id))
    },
    onError: (err) => {
      setCreateError(err instanceof ApiError ? err.message : 'Failed to create organization')
    },
  })

  const handleCreate = () => {
    setCreateError('')
    createMutation.mutate(name)
  }

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h5" fontWeight={700}>Organizations</Typography>
        <Button startIcon={<AddIcon />} variant="contained" onClick={() => setCreateOpen(true)}>
          New organization
        </Button>
      </Box>

      {isLoading && <CircularProgress />}
      {error && <Alert severity="error">{(error as ApiError).message}</Alert>}

      {orgs && orgs.length === 0 && (
        <Typography color="text.secondary">
          You don&apos;t belong to any organizations yet. Create one to get started.
        </Typography>
      )}

      <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(260px, 1fr))', gap: 2 }}>
        {orgs?.map((org) => (
          <Card key={org.id} variant="outlined">
            <CardActionArea onClick={() => navigate(ROUTES.ORG_DETAIL.replace(':id', org.id))}>
              <CardContent>
                <Typography variant="h6">{org.name}</Typography>
                <Typography variant="body2" color="text.secondary">/{org.slug}</Typography>
              </CardContent>
            </CardActionArea>
          </Card>
        ))}
      </Box>

      <Dialog open={createOpen} onClose={() => setCreateOpen(false)} maxWidth="xs" fullWidth>
        <DialogTitle>Create organization</DialogTitle>
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
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateOpen(false)}>Cancel</Button>
          <Button
            onClick={handleCreate}
            variant="contained"
            disabled={!name.trim() || createMutation.isPending}
          >
            {createMutation.isPending ? 'Creating…' : 'Create'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}
