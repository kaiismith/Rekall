import { useEffect, useState } from 'react'
import { Link as RouterLink, useNavigate, useSearchParams } from 'react-router-dom'
import { Alert, Box, Button, CircularProgress, Stack, Typography } from '@mui/material'
import { organizationService } from '@/services/organizationService'
import { useAuthStore } from '@/store/authStore'
import { ApiError } from '@/services/api'
import { ROUTES } from '@/constants'
import { ActionCard, GradientButton, HeroHeader } from '@/components/common/ui'

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

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', minHeight: '100vh' }}>
      <Box
        sx={{
          flex: 1,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          px: { xs: 2, sm: 3 },
          py: { xs: 4, sm: 8 },
        }}
      >
        <Stack spacing={4} alignItems="center" sx={{ width: '100%', maxWidth: 440 }}>
          {state === 'unauthenticated' && (
            <>
              <HeroHeader
                title="Sign in to accept"
                subtitle="You need to be signed in to accept this invitation."
              />
              <ActionCard maxWidth={440}>
                <Stack spacing={2}>
                  <GradientButton
                    onClick={() =>
                      navigate(ROUTES.LOGIN, {
                        state: { from: { pathname: `/invitations/accept`, search: `?token=${token}` } },
                      })
                    }
                  >
                    Sign in
                  </GradientButton>
                  <Button component={RouterLink} to={ROUTES.REGISTER} variant="outlined" fullWidth>
                    Create an account
                  </Button>
                </Stack>
              </ActionCard>
            </>
          )}

          {state === 'loading' && (
            <ActionCard maxWidth={440}>
              <Stack spacing={2} alignItems="center" py={3}>
                <CircularProgress size={28} />
                <Typography color="text.secondary">Accepting invitation…</Typography>
              </Stack>
            </ActionCard>
          )}

          {state === 'success' && (
            <>
              <HeroHeader
                title="Welcome aboard"
                subtitle={`You've joined ${orgName}.`}
              />
              <ActionCard maxWidth={440}>
                <Stack spacing={2}>
                  <Alert severity="success">
                    You&apos;re now a member of <strong>{orgName}</strong>.
                  </Alert>
                  <GradientButton
                    onClick={() => navigate(ROUTES.ORGANIZATIONS, { replace: true })}
                  >
                    Go to organizations
                  </GradientButton>
                </Stack>
              </ActionCard>
            </>
          )}

          {state === 'error' && (
            <>
              <HeroHeader
                title="Invitation error"
                subtitle="We couldn't process this invitation."
              />
              <ActionCard maxWidth={440}>
                <Stack spacing={2}>
                  <Alert severity="error">{errorMessage}</Alert>
                  <Button component={RouterLink} to={ROUTES.DASHBOARD} variant="outlined" fullWidth>
                    Go to dashboard
                  </Button>
                </Stack>
              </ActionCard>
            </>
          )}
        </Stack>
      </Box>
    </Box>
  )
}
