import {
  Box,
  Card,
  CardContent,
  Stack,
  Switch,
  ToggleButton,
  ToggleButtonGroup,
  Typography,
} from '@mui/material'
import type { ReactNode } from 'react'
import { PageHeader } from '@/components/common/ui'
import { useUIPreferencesStore } from '@/store/uiPreferencesStore'

function PreferenceRow({
  label,
  helper,
  control,
}: {
  label: string
  helper: string
  control: ReactNode
}) {
  return (
    <Card>
      <CardContent sx={{ p: 3 }}>
        <Stack
          direction={{ xs: 'column', sm: 'row' }}
          alignItems={{ xs: 'flex-start', sm: 'center' }}
          justifyContent="space-between"
          spacing={{ xs: 2, sm: 3 }}
        >
          <Box sx={{ minWidth: 0, maxWidth: 620 }}>
            <Typography
              variant="subtitle1"
              sx={{ fontWeight: 600, letterSpacing: '-0.005em', color: 'text.primary' }}
            >
              {label}
            </Typography>
            <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
              {helper}
            </Typography>
          </Box>
          <Box sx={{ flexShrink: 0 }}>{control}</Box>
        </Stack>
      </CardContent>
    </Card>
  )
}

export function SettingsPage() {
  const {
    sidebarDefault,
    reducedMotion,
    keyboardShortcutsEnabled,
    setSidebarDefault,
    setReducedMotion,
    setKeyboardShortcutsEnabled,
  } = useUIPreferencesStore()

  return (
    <Box>
      <PageHeader
        title="Settings"
        subtitle="Preferences that apply to this browser."
      />

      <Stack spacing={2}>
        <PreferenceRow
          label="Sidebar default state"
          helper="Whether the sidebar opens expanded or collapsed on your next visit."
          control={
            <ToggleButtonGroup
              value={sidebarDefault}
              exclusive
              size="small"
              onChange={(_, v) => v && setSidebarDefault(v)}
              aria-label="Sidebar default state"
            >
              <ToggleButton value="expanded" sx={{ textTransform: 'none', px: 2 }}>
                Expanded
              </ToggleButton>
              <ToggleButton value="collapsed" sx={{ textTransform: 'none', px: 2 }}>
                Collapsed
              </ToggleButton>
            </ToggleButtonGroup>
          }
        />

        <PreferenceRow
          label="Reduce motion"
          helper="Disable most in-app transitions and animations. Honours this setting in addition to the OS-level preference."
          control={
            <Switch
              checked={reducedMotion}
              onChange={(_, v) => setReducedMotion(v)}
              inputProps={{ 'aria-label': 'Reduce motion' }}
            />
          }
        />

        <PreferenceRow
          label="Enable keyboard shortcuts"
          helper="Turns on Ctrl/⌘ + Shift + C to start a new meeting from anywhere on the landing page."
          control={
            <Switch
              checked={keyboardShortcutsEnabled}
              onChange={(_, v) => setKeyboardShortcutsEnabled(v)}
              inputProps={{ 'aria-label': 'Enable keyboard shortcuts' }}
            />
          }
        />
      </Stack>

      <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 3, textAlign: 'center' }}>
        Preferences are stored in this browser only.
      </Typography>
    </Box>
  )
}
