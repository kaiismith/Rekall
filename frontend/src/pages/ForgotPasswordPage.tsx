import { useState } from 'react'
import { Link as RouterLink } from 'react-router-dom'
import { Alert, Box, Button, Link, Stack, TextField, Typography } from '@mui/material'
import { authService } from '@/services/authService'
import { ROUTES } from '@/constants'
import { ActionCard, GradientButton, HeroHeader } from '@/components/common/ui'

export function ForgotPasswordPage() {
  const [email, setEmail] = useState('')
  const [submitted, setSubmitted] = useState(false)
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    try {
      await authService.forgotPassword({ email })
    } catch {
      // Silently swallowed — anti-enumeration: never reveal whether the email exists.
    } finally {
      setSubmitted(true)
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

  if (submitted) {
    return (
      <Box sx={containerSx}>
        <Box sx={innerSx}>
          <Stack spacing={4} alignItems="center" sx={{ width: '100%', maxWidth: 440 }}>
            <HeroHeader
              title="Check your inbox"
              subtitle="If that email matches an account, a reset link is on its way."
            />
            <ActionCard maxWidth={440}>
              <Stack spacing={2}>
                <Alert severity="success">
                  If an account with that email exists, we sent a password reset link.
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
    <Box sx={containerSx}>
      <Box sx={innerSx}>
        <Stack spacing={4} alignItems="center" sx={{ width: '100%', maxWidth: 440 }}>
          <HeroHeader
            title="Reset your password"
            subtitle="Enter your email and we'll send you a link to set a new password."
          />
          <ActionCard maxWidth={440}>
            <Box
              component="form"
              onSubmit={handleSubmit}
              sx={{ display: 'flex', flexDirection: 'column', gap: 2.5 }}
            >
              <TextField
                label="Email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                fullWidth
                autoFocus
                autoComplete="email"
              />
              <GradientButton type="submit" disabled={loading}>
                {loading ? 'Sending…' : 'Send reset link'}
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
