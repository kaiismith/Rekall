import { useEffect, useState } from 'react'
import { Link as RouterLink, useSearchParams } from 'react-router-dom'
import { Alert, Box, Button, CircularProgress, Container, Typography } from '@mui/material'
import { authService } from '@/services/authService'
import { ApiError } from '@/services/api'
import { ROUTES } from '@/constants'

type State = 'loading' | 'success' | 'error'

export function VerifyEmailPage() {
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
    <Container maxWidth="xs">
      <Box sx={{ mt: 8, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 3 }}>
        {state === 'loading' && (
          <>
            <CircularProgress />
            <Typography>Verifying your email…</Typography>
          </>
        )}

        {state === 'success' && (
          <>
            <Typography variant="h5" fontWeight={700}>Email verified!</Typography>
            <Alert severity="success" sx={{ width: '100%' }}>
              Your email address has been confirmed. You can now sign in.
            </Alert>
            <Button component={RouterLink} to={ROUTES.LOGIN} variant="contained" fullWidth>
              Sign in
            </Button>
          </>
        )}

        {state === 'error' && (
          <>
            <Typography variant="h5" fontWeight={700}>Verification failed</Typography>
            <Alert severity="error" sx={{ width: '100%' }}>{errorMessage}</Alert>
            <Button component={RouterLink} to={ROUTES.LOGIN} variant="outlined" fullWidth>
              Back to sign in
            </Button>
          </>
        )}
      </Box>
    </Container>
  )
}
