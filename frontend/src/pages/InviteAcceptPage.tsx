import { useEffect, useState } from 'react'
import { Link as RouterLink, useNavigate, useSearchParams } from 'react-router-dom'
import { Alert, Box, Button, CircularProgress, Container, Typography } from '@mui/material'
import { organizationService } from '@/services/organizationService'
import { useAuthStore } from '@/store/authStore'
import { ApiError } from '@/services/api'
import { ROUTES } from '@/constants'

type State = 'loading' | 'success' | 'error' | 'unauthenticated'

export function InviteAcceptPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token') ?? ''
  const { accessToken } = useAuthStore()

  const [state, setState] = useState<State>(accessToken ? 'loading' : 'unauthenticated')
  const [errorMessage, setErrorMessage] = useState('')
  const [orgName, setOrgName] = useState('')

  useEffect(() => {
    if (!accessToken || !token) return

    organizationService
      .acceptInvitation({ token })
      .then((org) => {
        setOrgName(org.name)
        setState('success')
      })
      .catch((err) => {
        setErrorMessage(err instanceof ApiError ? err.message : 'Failed to accept invitation.')
        setState('error')
      })
  }, [token, accessToken])

  if (state === 'unauthenticated') {
    return (
      <Container maxWidth="xs">
        <Box sx={{ mt: 8, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 3 }}>
          <Typography variant="h5" fontWeight={700}>Sign in to accept</Typography>
          <Alert severity="info" sx={{ width: '100%' }}>
            You need to be signed in to accept this invitation.
          </Alert>
          <Button
            component={RouterLink}
            to={ROUTES.LOGIN}
            state={{ from: { pathname: `/invitations/accept`, search: `?token=${token}` } }}
            variant="contained"
            fullWidth
          >
            Sign in
          </Button>
          <Button component={RouterLink} to={ROUTES.REGISTER} variant="outlined" fullWidth>
            Create an account
          </Button>
        </Box>
      </Container>
    )
  }

  return (
    <Container maxWidth="xs">
      <Box sx={{ mt: 8, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 3 }}>
        {state === 'loading' && (
          <>
            <CircularProgress />
            <Typography>Accepting invitation…</Typography>
          </>
        )}

        {state === 'success' && (
          <>
            <Typography variant="h5" fontWeight={700}>Welcome aboard!</Typography>
            <Alert severity="success" sx={{ width: '100%' }}>
              You&apos;ve joined <strong>{orgName}</strong>.
            </Alert>
            <Button
              variant="contained"
              fullWidth
              onClick={() => navigate(ROUTES.ORGANIZATIONS, { replace: true })}
            >
              Go to organizations
            </Button>
          </>
        )}

        {state === 'error' && (
          <>
            <Typography variant="h5" fontWeight={700}>Invitation error</Typography>
            <Alert severity="error" sx={{ width: '100%' }}>{errorMessage}</Alert>
            <Button component={RouterLink} to={ROUTES.DASHBOARD} variant="outlined" fullWidth>
              Go to dashboard
            </Button>
          </>
        )}
      </Box>
    </Container>
  )
}
