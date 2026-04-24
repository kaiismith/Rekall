import { useState } from 'react'
import { Link as RouterLink } from 'react-router-dom'
import {
  Alert,
  Box,
  Button,
  Link,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import { authService } from '@/services/authService'
import { ApiError } from '@/services/api'
import { ROUTES } from '@/constants'
import { ActionCard, GradientButton, HeroHeader, PasswordField } from '@/components/common/ui'

export function RegisterPage() {
  const [fullName, setFullName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)
  const [loading, setLoading] = useState(false)

  const passwordsMatch = password === confirmPassword
  const showMismatch = confirmPassword.length > 0 && !passwordsMatch
  const canSubmit =
    fullName.trim().length > 0 &&
    email.trim().length > 0 &&
    password.length >= 8 &&
    passwordsMatch &&
    !loading

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!canSubmit) return
    setError('')
    setLoading(true)
    try {
      await authService.register({ email, password, full_name: fullName })
      setSuccess(true)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'An unexpected error occurred')
    } finally {
      setLoading(false)
    }
  }

  if (success) {
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
            <HeroHeader title="Check your inbox" />
            <ActionCard maxWidth={440}>
              <Stack spacing={3}>
                <Alert severity="success">
                  We sent a verification link to <strong>{email}</strong>. Click it to activate
                  your account.
                </Alert>
                <Button component={RouterLink} to={ROUTES.LOGIN} variant="outlined" fullWidth>
                  Back to sign in
                </Button>
              </Stack>
            </ActionCard>
          </Stack>
        </Box>
      </Box>
    )
  }

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
          <HeroHeader title="Create your account" />

          <ActionCard maxWidth={440}>
            <Box
              component="form"
              onSubmit={handleSubmit}
              noValidate
              sx={{ display: 'flex', flexDirection: 'column', gap: 2.5 }}
            >
              {error && <Alert severity="error">{error}</Alert>}
              <Stack spacing={2}>
                <TextField
                  label="Full name"
                  value={fullName}
                  onChange={(e) => setFullName(e.target.value)}
                  required
                  fullWidth
                  autoFocus
                  autoComplete="name"
                />
                <TextField
                  label="Email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  fullWidth
                  autoComplete="email"
                />
                <PasswordField
                  label="Password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  fullWidth
                  autoComplete="new-password"
                  helperText="At least 8 characters with a letter and a digit"
                />
                <PasswordField
                  label="Confirm password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  required
                  fullWidth
                  autoComplete="new-password"
                  error={showMismatch}
                  helperText={showMismatch ? 'Passwords do not match.' : ' '}
                />
              </Stack>
              <GradientButton type="submit" disabled={!canSubmit} sx={{ mt: 0.5 }}>
                {loading ? 'Creating account…' : 'Create account'}
              </GradientButton>
            </Box>
          </ActionCard>

          <Typography variant="body2">
            Already have an account?{' '}
            <Link component={RouterLink} to={ROUTES.LOGIN}>
              Sign in
            </Link>
          </Typography>
        </Stack>
      </Box>
    </Box>
  )
}
