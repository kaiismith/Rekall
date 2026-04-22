import { useState } from 'react'
import { Link as RouterLink } from 'react-router-dom'
import { Alert, Box, Button, Container, Link, TextField, Typography } from '@mui/material'
import { authService } from '@/services/authService'
import { ROUTES } from '@/constants'

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

  if (submitted) {
    return (
      <Container maxWidth="xs">
        <Box sx={{ mt: 8, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 3 }}>
          <Typography variant="h5" fontWeight={700}>Check your inbox</Typography>
          <Alert severity="success" sx={{ width: '100%' }}>
            If an account with that email exists, we sent a password reset link.
          </Alert>
          <Button component={RouterLink} to={ROUTES.LOGIN} variant="outlined" fullWidth>
            Back to sign in
          </Button>
        </Box>
      </Container>
    )
  }

  return (
    <Container maxWidth="xs">
      <Box sx={{ mt: 8, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 3 }}>
        <Typography variant="h4" fontWeight={700}>Reset password</Typography>
        <Typography variant="body2" color="text.secondary" textAlign="center">
          Enter your email and we&apos;ll send you a link to reset your password.
        </Typography>

        <Box component="form" onSubmit={handleSubmit} sx={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 2 }}>
          <TextField
            label="Email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            fullWidth
            autoFocus
          />
          <Button type="submit" variant="contained" size="large" fullWidth disabled={loading}>
            {loading ? 'Sending…' : 'Send reset link'}
          </Button>
        </Box>

        <Link component={RouterLink} to={ROUTES.LOGIN} variant="body2">
          Back to sign in
        </Link>
      </Box>
    </Container>
  )
}
