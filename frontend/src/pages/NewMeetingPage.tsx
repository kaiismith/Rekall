import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import {
  Alert,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  InputAdornment,
  MenuItem,
  Select,
  Stack,
  Switch,
  TextField,
  Typography,
} from '@mui/material'
import VideocamOutlinedIcon from '@mui/icons-material/VideocamOutlined'
import KeyOutlinedIcon from '@mui/icons-material/KeyOutlined'
import WarningAmberRoundedIcon from '@mui/icons-material/WarningAmberRounded'
import { meetingService } from '@/services/meetingService'
import { MIN_MEETING_CODE_LENGTH, ROUTES } from '@/constants'
import { useUIPreferencesStore } from '@/store/uiPreferencesStore'
import { useOrgsStore } from '@/store/orgsStore'
import { useDeptsStore } from '@/store/deptsStore'
import { ApiError } from '@/services/api'
import {
  ActionCard,
  BackToHomeLink,
  GradientButton,
  HeroHeader,
  OrDivider,
  ScopePicker,
  ShortcutHint,
} from '@/components/common/ui'
import type { Scope } from '@/types/scope'
import { parseScopeFromUrl } from '@/utils/scope'

type TranscriptLanguage = 'en' | 'es' | 'fr' | 'de' | 'ja' | 'zh'

const LANGUAGE_OPTIONS: Array<{ value: TranscriptLanguage; label: string }> = [
  { value: 'en', label: 'English' },
  { value: 'es', label: 'Español' },
  { value: 'fr', label: 'Français' },
  { value: 'de', label: 'Deutsch' },
  { value: 'ja', label: '日本語' },
  { value: 'zh', label: '中文' },
]

interface CreateError {
  title: string
  message: string
  /** When true, render a "Manage meetings" CTA so the user can clean up. */
  showManageCTA?: boolean
}

export function NewMeetingPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const keyboardShortcutsEnabled = useUIPreferencesStore((s) => s.keyboardShortcutsEnabled)
  const [transcriptLanguage, setTranscriptLanguage] = useState<TranscriptLanguage>('en')
  const [code, setCode] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<CreateError | null>(null)

  // Scope picker — pre-fills from `?scope=` (set by scoped Meetings pages'
  // "New Meeting" button). Defaults to null (Personal/open).
  const initialScope = useMemo(() => parseScopeFromUrl(searchParams), [searchParams])
  const [scope, setScope] = useState<Scope | null>(initialScope)
  const [isPrivate, setIsPrivate] = useState(false)

  // Hide the picker entirely when the user has zero orgs — there is nothing
  // for them to pick.
  const orgs = useOrgsStore((s) => s.orgs)
  const loadOrgs = useOrgsStore((s) => s.load)
  const ensureDeptsLoaded = useDeptsStore((s) => s.ensureLoaded)
  useEffect(() => {
    if (orgs === null) void loadOrgs()
  }, [orgs, loadOrgs])
  useEffect(() => {
    if (initialScope?.type === 'department') void ensureDeptsLoaded(initialScope.orgId)
  }, [initialScope, ensureDeptsLoaded])

  const showScopePicker = (orgs?.length ?? 0) > 0

  const handleCreate = useCallback(async () => {
    if (loading) return
    setError(null)
    setLoading(true)
    try {
      const payload: Parameters<typeof meetingService.create>[0] = {
        title: '',
        type: scope && isPrivate ? 'private' : 'open',
      }
      if (scope?.type === 'organization') {
        payload.scope_type = 'organization'
        payload.scope_id = scope.id
      } else if (scope?.type === 'department') {
        payload.scope_type = 'department'
        payload.scope_id = scope.id
      }
      const res = await meetingService.create(payload)
      navigate(`/meeting/${res.data.code}`)
    } catch (e: unknown) {
      // Map known server errors to friendly modals; fall back to a generic
      // message for anything we don't recognise.
      if (e instanceof ApiError) {
        const isLimit =
          e.status === 400 && /maximum.*active meetings/i.test(e.message)
        if (isLimit) {
          setError({
            title: 'Meeting limit reached',
            message:
              'You\'ve hit the maximum number of active meetings. End one of your existing meetings before creating a new one.',
            showManageCTA: true,
          })
        } else {
          setError({
            title: 'Couldn\'t create the meeting',
            message: e.message,
          })
        }
      } else {
        const msg = (e as { response?: { data?: { error?: { message?: string } } } })
          ?.response?.data?.error?.message
        setError({
          title: 'Couldn\'t create the meeting',
          message: msg ?? 'Something went wrong. Please try again.',
        })
      }
    } finally {
      setLoading(false)
    }
  }, [navigate, loading, scope, isPrivate])

  // Ctrl/⌘ + Shift + C shortcut — create a meeting from anywhere on this page.
  // Gated on the user's UI preference so it can be disabled from Settings.
  useEffect(() => {
    if (!keyboardShortcutsEnabled) return
    const handler = (e: KeyboardEvent) => {
      if (loading) return
      const target = e.target as HTMLElement | null
      if (
        target &&
        (target.tagName === 'INPUT' ||
          target.tagName === 'TEXTAREA' ||
          target.isContentEditable)
      ) {
        return
      }
      const oneOfCtrlOrMeta = (e.ctrlKey && !e.metaKey) || (e.metaKey && !e.ctrlKey)
      if (e.shiftKey && oneOfCtrlOrMeta && (e.key === 'C' || e.key === 'c')) {
        e.preventDefault()
        void handleCreate()
      }
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [handleCreate, loading, keyboardShortcutsEnabled])

  const canJoin = code.length >= MIN_MEETING_CODE_LENGTH
  const handleJoin = () => {
    if (canJoin) navigate(`/meeting/${code}`)
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', minHeight: '100vh' }}>
      <Box sx={{ p: { xs: 2, sm: 3 } }}>
        <BackToHomeLink />
      </Box>

      <Box
        sx={{
          flex: 1,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          px: { xs: 2, sm: 3 },
          pb: { xs: 4, sm: 8 },
        }}
      >
        <Stack spacing={4} alignItems="center" sx={{ width: '100%', maxWidth: 520 }}>
          <HeroHeader
            title="Rekall Meeting"
            subtitle="Create a new meeting or join with a code."
          />

          <ActionCard>
            <Stack spacing={2}>
              {showScopePicker && (
                <Stack spacing={1}>
                  <Typography variant="body2" color="text.secondary">
                    Scope
                  </Typography>
                  <ScopePicker value={scope} onChange={setScope} />
                  {scope && (scope.type === 'organization' || scope.type === 'department') && (
                    <FormControlLabel
                      control={
                        <Switch
                          size="small"
                          checked={isPrivate}
                          onChange={(_, v) => setIsPrivate(v)}
                        />
                      }
                      label={
                        <Typography variant="body2" color="text.secondary">
                          Private (members of this scope only)
                        </Typography>
                      }
                    />
                  )}
                </Stack>
              )}

              <Typography variant="body2" color="text.secondary">
                Transcript language
              </Typography>
              <Select
                value={transcriptLanguage}
                onChange={(e) => setTranscriptLanguage(e.target.value as TranscriptLanguage)}
                fullWidth
                size="small"
              >
                {LANGUAGE_OPTIONS.map((opt) => (
                  <MenuItem key={opt.value} value={opt.value}>
                    {opt.label}
                  </MenuItem>
                ))}
              </Select>

              <GradientButton
                startIcon={<VideocamOutlinedIcon />}
                onClick={handleCreate}
                disabled={loading}
              >
                {loading ? 'Creating…' : 'Create meeting'}
              </GradientButton>

              {keyboardShortcutsEnabled && <ShortcutHint keys={['Ctrl/⌘', 'Shift', 'C']} />}

              <OrDivider />

              <Typography variant="overline" color="text.secondary">
                JOIN A MEETING
              </Typography>
              <Typography variant="body2">Enter the code shared by the host.</Typography>

              <TextField
                fullWidth
                placeholder="Enter meeting code"
                value={code}
                onChange={(e) => setCode(e.target.value.trim().toLowerCase())}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') handleJoin()
                }}
                inputProps={{
                  'aria-label': 'Meeting code',
                  maxLength: 32,
                  spellCheck: false,
                  autoComplete: 'off',
                }}
                InputProps={{
                  startAdornment: (
                    <InputAdornment position="start">
                      <KeyOutlinedIcon fontSize="small" />
                    </InputAdornment>
                  ),
                }}
              />

              <Button
                variant="outlined"
                fullWidth
                disabled={!canJoin}
                onClick={handleJoin}
              >
                Join
              </Button>
            </Stack>
          </ActionCard>
        </Stack>
      </Box>

      {/* Create-meeting error modal — surfaces the meeting-limit error in a
          prominent, actionable way instead of a small line of red text. */}
      <Dialog
        open={!!error}
        onClose={() => setError(null)}
        maxWidth="xs"
        fullWidth
        PaperProps={{
          sx: {
            borderRadius: '14px',
            border: '1px solid rgba(245,158,11,0.25)',
          },
        }}
      >
        <DialogTitle sx={{ pb: 1.5, pt: 3 }}>
          <Stack direction="row" spacing={1.5} alignItems="center">
            <Box
              sx={{
                width: 36,
                height: 36,
                borderRadius: '10px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                bgcolor: 'rgba(245,158,11,0.12)',
                color: '#fcd34d',
                flexShrink: 0,
              }}
            >
              <WarningAmberRoundedIcon fontSize="small" />
            </Box>
            <Typography
              component="span"
              sx={{ fontWeight: 600, fontSize: '1.0625rem', letterSpacing: '-0.01em' }}
            >
              {error?.title}
            </Typography>
          </Stack>
        </DialogTitle>

        <DialogContent sx={{ pt: '8px !important' }}>
          <Alert severity="warning" icon={false} sx={{ mb: 0 }}>
            {error?.message}
          </Alert>
        </DialogContent>

        <DialogActions sx={{ px: 3, pb: 2.5, pt: 1, gap: 1 }}>
          <Button onClick={() => setError(null)} color="inherit">
            Dismiss
          </Button>
          {error?.showManageCTA && (
            <GradientButton
              fullWidth={false}
              onClick={() => {
                setError(null)
                navigate(ROUTES.MEETINGS)
              }}
            >
              Manage meetings
            </GradientButton>
          )}
        </DialogActions>
      </Dialog>
    </Box>
  )
}
