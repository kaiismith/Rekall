import { useState } from 'react'
import { Link as RouterLink, useNavigate, useSearchParams } from 'react-router-dom'
import { Alert, Box, Link, Stack, Typography } from '@mui/material'
import { authService } from '@/services/authService'
import { ApiError } from '@/services/api'
import { ROUTES } from '@/constants'
import { ActionCard, GradientButton, HeroHeader, PasswordField } from '@/components/common/ui'

export function ResetPasswordPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token') ?? ''

  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await authService.resetPassword({ token, password })
      navigate(ROUTES.LOGIN, { replace: true, state: { passwordReset: true } })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'An unexpected error occurred')
    } finally {
      setLoading(false)
    }
  }

  const containerSx = { display: 'flex', flexDirection: 'column', minHeight: '100vh' } as const
  const innerSx = {
    flex: 1,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    px: { xs: 2, sm: 3 },
    py: { xs: 4, sm: 8 },
  }

  if (!token) {
    return (
      <Box sx={containerSx}>
        <Box sx={innerSx}>
          <Stack spacing={4} alignItems="center" sx={{ width: '100%', maxWidth: 440 }}>
            <HeroHeader
              title="Invalid reset link"
              subtitle="This link is no longer valid."
            />
            <ActionCard maxWidth={440}>
              <Stack spacing={2}>
                <Alert severity="error">This reset link is invalid or has expired.</Alert>
                <Link
                  component={RouterLink}
                  to={ROUTES.FORGOT_PASSWORD}
                  variant="body2"
                  underline="hover"
                  sx={{ textAlign: 'center' }}
                >
                  Request a new reset link
                </Link>
              </Stack>
            </ActionCard>
          </Stack>
        </Box>
      </Box>
    )
  }

  return (
    <Box sx={containerSx}>
      <Box sx={innerSx}>
        <Stack spacing={4} alignItems="center" sx={{ width: '100%', maxWidth: 440 }}>
          <HeroHeader
            title="Set a new password"
            subtitle="Choose a strong password you haven't used before."
          />
          <ActionCard maxWidth={440}>
            <Box
              component="form"
              onSubmit={handleSubmit}
              sx={{ display: 'flex', flexDirection: 'column', gap: 2.5 }}
            >
              {error && <Alert severity="error">{error}</Alert>}
              <PasswordField
                label="New password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                fullWidth
                autoFocus
                autoComplete="new-password"
                helperText="At least 8 characters with a letter and a digit"
              />
              <GradientButton type="submit" disabled={loading}>
                {loading ? 'Saving…' : 'Set new password'}
              </GradientButton>
            </Box>
          </ActionCard>
          <Typography variant="body2" color="text.secondary">
            Remembered it?{' '}
            <Link component={RouterLink} to={ROUTES.LOGIN} underline="hover">
              Back to sign in
            </Link>
          </Typography>
        </Stack>
      </Box>
    </Box>
  )
}
