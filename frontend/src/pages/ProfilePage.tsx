import { useEffect, useState } from 'react'
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import CheckCircleRoundedIcon from '@mui/icons-material/CheckCircleRounded'
import WarningAmberRoundedIcon from '@mui/icons-material/WarningAmberRounded'
import { useAuthStore } from '@/store/authStore'
import { authService } from '@/services/authService'
import { ApiError } from '@/services/api'
import { GradientButton, PageHeader, PasswordField } from '@/components/common/ui'
import { tokens } from '@/theme'

const MAX_NAME_LENGTH = 100

function IdentityCard({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <Box
      sx={{
        display: 'grid',
        gridTemplateColumns: { xs: '1fr', sm: '180px 1fr' },
        gap: { xs: 0.5, sm: 3 },
        py: 1.5,
        borderBottom: '1px solid rgba(255,255,255,0.04)',
        '&:last-of-type': { borderBottom: 0 },
        alignItems: 'center',
      }}
    >
      <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 500 }}>
        {label}
      </Typography>
      <Box sx={{ color: 'text.primary', fontSize: '0.9375rem' }}>{children}</Box>
    </Box>
  )
}

function formatDate(iso?: string): string {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  return d.toLocaleDateString(undefined, { year: 'numeric', month: 'long', day: 'numeric' })
}

function DisplayNameSection() {
  const { user, accessToken, setAuth } = useAuthStore()
  const [editing, setEditing] = useState(false)
  const [value, setValue] = useState(user?.full_name ?? '')
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [justSaved, setJustSaved] = useState(false)

  useEffect(() => {
    setValue(user?.full_name ?? '')
  }, [user?.full_name])

  if (!user) return null

  const trimmed = value.trim()
  const tooLong = trimmed.length > MAX_NAME_LENGTH
  const unchanged = trimmed === user.full_name
  const canSave = !saving && !tooLong && trimmed.length > 0 && !unchanged

  const handleSave = async () => {
    if (!canSave) return
    setSaving(true)
    setError(null)
    try {
      const updated = await authService.updateMe({ full_name: trimmed })
      if (accessToken) setAuth(updated, accessToken)
      setEditing(false)
      setJustSaved(true)
      setTimeout(() => setJustSaved(false), 3000)
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Failed to update display name.')
    } finally {
      setSaving(false)
    }
  }

  const handleCancel = () => {
    setValue(user.full_name)
    setError(null)
    setEditing(false)
  }

  return (
    <Card>
      <CardContent sx={{ p: 3 }}>
        <Typography
          variant="overline"
          color="text.secondary"
          sx={{ fontWeight: 700, letterSpacing: '0.12em', display: 'block', mb: 1.5 }}
        >
          Display name
        </Typography>

        {justSaved && <Alert severity="success" sx={{ mb: 2 }}>Display name updated.</Alert>}
        {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5} alignItems="flex-start">
          <TextField
            value={value}
            onChange={(e) => setValue(e.target.value)}
            fullWidth
            disabled={!editing || saving}
            autoFocus={editing}
            error={tooLong}
            helperText={
              editing
                ? tooLong
                  ? `Must be ${MAX_NAME_LENGTH} characters or fewer.`
                  : `${trimmed.length} / ${MAX_NAME_LENGTH}`
                : 'Shown to other participants across meetings and chat.'
            }
            inputProps={{ maxLength: MAX_NAME_LENGTH + 1, 'aria-label': 'Display name' }}
          />

          {editing ? (
            <Stack direction="row" spacing={1} sx={{ pt: { xs: 1, sm: 0.5 } }}>
              <Button variant="outlined" onClick={handleCancel} disabled={saving}>
                Cancel
              </Button>
              <GradientButton
                fullWidth={false}
                onClick={handleSave}
                disabled={!canSave}
                sx={{ minWidth: 96 }}
              >
                {saving ? 'Saving…' : 'Save'}
              </GradientButton>
            </Stack>
          ) : (
            <Button
              variant="outlined"
              onClick={() => setEditing(true)}
              sx={{ pt: { sm: 0.5 }, flexShrink: 0 }}
            >
              Edit
            </Button>
          )}
        </Stack>
      </CardContent>
    </Card>
  )
}

function SecuritySection() {
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirmNext, setConfirmNext] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [currentError, setCurrentError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [justSaved, setJustSaved] = useState(false)

  const { user, accessToken, setAuth } = useAuthStore()

  const newPasswordIssue = validateNewPassword(next)
  const confirmMismatch = confirmNext.length > 0 && confirmNext !== next
  const canSubmit =
    !saving &&
    current.length > 0 &&
    next.length > 0 &&
    !newPasswordIssue &&
    !confirmMismatch &&
    confirmNext === next

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!canSubmit || !user || !accessToken) return
    setSaving(true)
    setError(null)
    setCurrentError(null)
    try {
      const { access_token } = await authService.changePassword({
        current_password: current,
        new_password: next,
      })
      setAuth(user, access_token)
      setCurrent('')
      setNext('')
      setConfirmNext('')
      setJustSaved(true)
      setTimeout(() => setJustSaved(false), 5000)
    } catch (e) {
      if (e instanceof ApiError && e.code === 'INVALID_CURRENT_PASSWORD') {
        setCurrentError('Current password is incorrect.')
      } else {
        setError(e instanceof ApiError ? e.message : 'Failed to change password.')
      }
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card>
      <CardContent sx={{ p: 3 }}>
        <Typography
          variant="overline"
          color="text.secondary"
          sx={{ fontWeight: 700, letterSpacing: '0.12em', display: 'block', mb: 1.5 }}
        >
          Security
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          Change your password. Any other devices currently signed in will be
          signed out as a precaution.
        </Typography>

        {justSaved && (
          <Alert severity="success" sx={{ mb: 2 }}>
            Password updated. Other active sessions have been signed out.
          </Alert>
        )}
        {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

        <Box component="form" onSubmit={handleSubmit} noValidate>
          <Stack spacing={2}>
            <PasswordField
              label="Current password"
              value={current}
              onChange={(e) => {
                setCurrent(e.target.value)
                if (currentError) setCurrentError(null)
              }}
              required
              fullWidth
              autoComplete="current-password"
              error={!!currentError}
              helperText={currentError ?? ' '}
            />
            <PasswordField
              label="New password"
              value={next}
              onChange={(e) => setNext(e.target.value)}
              required
              fullWidth
              autoComplete="new-password"
              error={next.length > 0 && !!newPasswordIssue}
              helperText={
                next.length === 0
                  ? 'At least 8 characters with a letter and a digit.'
                  : newPasswordIssue ?? 'Strong enough.'
              }
            />
            <PasswordField
              label="Confirm new password"
              value={confirmNext}
              onChange={(e) => setConfirmNext(e.target.value)}
              required
              fullWidth
              autoComplete="new-password"
              error={confirmMismatch}
              helperText={confirmMismatch ? 'Passwords do not match.' : ' '}
            />
            <GradientButton type="submit" disabled={!canSubmit} sx={{ mt: 0.5 }}>
              {saving ? 'Saving…' : 'Change password'}
            </GradientButton>
          </Stack>
        </Box>
      </CardContent>
    </Card>
  )
}

function validateNewPassword(pw: string): string | null {
  if (pw.length === 0) return null
  if (pw.length < 8) return 'At least 8 characters.'
  if (!/[A-Za-z]/.test(pw)) return 'Must contain a letter.'
  if (!/\d/.test(pw)) return 'Must contain a digit.'
  return null
}

export function ProfilePage() {
  const { user } = useAuthStore()

  if (!user) {
    return (
      <Box display="flex" justifyContent="center" mt={10}>
        <CircularProgress />
      </Box>
    )
  }

  return (
    <Box>
      <PageHeader
        title="Profile"
        subtitle="Your account identity and security settings."
      />

      <Stack spacing={2.5}>
        <Card>
          <CardContent sx={{ p: 3 }}>
            <Typography
              variant="overline"
              color="text.secondary"
              sx={{ fontWeight: 700, letterSpacing: '0.12em', display: 'block', mb: 1 }}
            >
              Identity
            </Typography>

            <IdentityCard label="Display name">
              {user.full_name || '—'}
            </IdentityCard>
            <IdentityCard label="Email">
              <Box component="span" sx={{ fontFamily: tokens.fonts.mono, fontSize: '0.875rem' }}>
                {user.email}
              </Box>
            </IdentityCard>
            <IdentityCard label="Role">
              <Chip
                label={user.role}
                size="small"
                sx={{
                  textTransform: 'capitalize',
                  fontWeight: 600,
                  bgcolor: 'rgba(129,140,248,0.12)',
                  color: '#c4b5fd',
                  border: '1px solid rgba(129,140,248,0.25)',
                }}
              />
            </IdentityCard>
            <IdentityCard label="Email verified">
              {user.email_verified ? (
                <Stack direction="row" spacing={0.75} alignItems="center">
                  <CheckCircleRoundedIcon sx={{ fontSize: '1rem', color: '#22c55e' }} />
                  <Typography variant="body2" sx={{ color: '#86efac', fontWeight: 500 }}>
                    Verified
                  </Typography>
                </Stack>
              ) : (
                <Stack direction="row" spacing={0.75} alignItems="center">
                  <WarningAmberRoundedIcon sx={{ fontSize: '1rem', color: '#f59e0b' }} />
                  <Typography variant="body2" sx={{ color: '#fcd34d', fontWeight: 500 }}>
                    Not verified
                  </Typography>
                </Stack>
              )}
            </IdentityCard>
            <IdentityCard label="Member since">
              {formatDate(user.created_at)}
            </IdentityCard>
          </CardContent>
        </Card>

        <DisplayNameSection />
        <SecuritySection />
      </Stack>
    </Box>
  )
}
