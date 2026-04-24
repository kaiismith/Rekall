import { useState } from 'react'
import { Link as RouterLink, Navigate, useNavigate } from 'react-router-dom'
import {
  Alert,
  Box,
  Link,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import { useAuthStore } from '@/store/authStore'
import { authService } from '@/services/authService'
import { ApiError } from '@/services/api'
import { ROUTES } from '@/constants'
import { ProtectedSplash } from '@/components/common/ProtectedSplash'
import { ActionCard, GradientButton, HeroHeader, PasswordField } from '@/components/common/ui'

export function LoginPage() {
  const navigate = useNavigate()
  const { setAuth, accessToken, isInitialised } = useAuthStore()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const result = await authService.login({ email, password })
      setAuth(result.user, result.access_token)
      // Always land on the dashboard after sign-in.
      navigate(ROUTES.DASHBOARD, { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'An unexpected error occurred')
    } finally {
      setLoading(false)
    }
  }

  if (!isInitialised) return <ProtectedSplash />
  if (accessToken) return <Navigate to={ROUTES.DASHBOARD} replace />

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
          <HeroHeader title="Sign in to Rekall" subtitle="Welcome back." />

          <ActionCard maxWidth={440}>
            <Box
              component="form"
              onSubmit={handleSubmit}
              noValidate
              autoComplete="off"
              sx={{ display: 'flex', flexDirection: 'column', gap: 2.5 }}
            >
              {error && <Alert severity="error">{error}</Alert>}

              <Stack spacing={2}>
                <TextField
                  label="Email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  fullWidth
                  autoFocus
                  // autoComplete="off" alone is ignored by Chrome/Edge on login
                  // forms; "new-password" is the accepted escape hatch that
                  // disables saved-credential autofill across browsers.
                  autoComplete="new-password"
                  inputProps={{
                    'aria-label': 'Email address',
                    autoCorrect: 'off',
                    autoCapitalize: 'off',
                    spellCheck: false,
                  }}
                />

                <Box>
                  <PasswordField
                    label="Password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                    fullWidth
                    autoComplete="new-password"
                    inputProps={{ 'aria-label': 'Password' }}
                  />
                  <Box sx={{ display: 'flex', justifyContent: 'flex-end', mt: 1 }}>
                    <Link
                      component={RouterLink}
                      to={ROUTES.FORGOT_PASSWORD}
                      variant="body2"
                      underline="hover"
                    >
                      Forgot password?
                    </Link>
                  </Box>
                </Box>
              </Stack>

              <GradientButton type="submit" disabled={loading} sx={{ mt: 0.5 }}>
                {loading ? 'Signing in…' : 'Sign in'}
              </GradientButton>
            </Box>
          </ActionCard>

          <Typography variant="body2" color="text.secondary">
            No account?{' '}
            <Link component={RouterLink} to={ROUTES.REGISTER} underline="hover">
              Create one
            </Link>
          </Typography>
        </Stack>
      </Box>
    </Box>
  )
}
