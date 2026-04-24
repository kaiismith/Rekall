import { useEffect, useState } from 'react'
import { Link as RouterLink, useNavigate, useSearchParams } from 'react-router-dom'
import { Alert, Box, Button, CircularProgress, Stack, Typography } from '@mui/material'


import { authService } from '@/services/authService'
import { ApiError } from '@/services/api'
import { ROUTES } from '@/constants'
import { ActionCard, GradientButton, HeroHeader } from '@/components/common/ui'

type State = 'loading' | 'success' | 'error'

export function VerifyEmailPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token') ?? ''

  const [state, setState] = useState<State>('loading')
  const [errorMessage, setErrorMessage] = useState('')

  useEffect(() => {
    if (!token) {
      setErrorMessage('No verification token found in this link.')
      setState('error')
      return
    }

    authService
      .verifyEmail(token)
      .then(() => setState('success'))
      .catch((err) => {
        setErrorMessage(err instanceof ApiError ? err.message : 'Verification failed.')
        setState('error')
      })
  }, [token])

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
          {state === 'loading' && (
            <ActionCard maxWidth={440}>
              <Stack spacing={2} alignItems="center" py={3}>
                <CircularProgress size={28} />
                <Typography color="text.secondary">Verifying your email…</Typography>
              </Stack>
            </ActionCard>
          )}

          {state === 'success' && (
            <>
              <HeroHeader
                title="Email verified"
                subtitle="Your address has been confirmed. You can now sign in."
              />
              <ActionCard maxWidth={440}>
                <Stack spacing={2}>
                  <Alert severity="success">
                    Your email address has been confirmed.
                  </Alert>
                  <GradientButton onClick={() => navigate(ROUTES.LOGIN)}>
                    Sign in
                  </GradientButton>
                </Stack>
              </ActionCard>
            </>
          )}

          {state === 'error' && (
            <>
              <HeroHeader
                title="Verification failed"
                subtitle="This link may have expired or already been used."
              />
              <ActionCard maxWidth={440}>
                <Stack spacing={2}>
                  <Alert severity="error">{errorMessage}</Alert>
                  <Button component={RouterLink} to={ROUTES.LOGIN} variant="outlined" fullWidth>
                    Back to sign in
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
