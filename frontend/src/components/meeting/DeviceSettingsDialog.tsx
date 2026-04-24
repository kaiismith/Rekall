import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  Typography,
  type SelectChangeEvent,
} from '@mui/material'
import VideocamOutlinedIcon from '@mui/icons-material/VideocamOutlined'
import MicOutlinedIcon from '@mui/icons-material/MicOutlined'
import VolumeUpOutlinedIcon from '@mui/icons-material/VolumeUpOutlined'
import type { ReactNode } from 'react'

interface DeviceSettingsDialogProps {
  open: boolean
  onClose: () => void
  cameras: MediaDeviceInfo[]
  mics: MediaDeviceInfo[]
  speakers: MediaDeviceInfo[]
  selectedCameraId: string
  selectedMicId: string
  selectedSpeakerId: string
  onSwitchCamera: (id: string) => void | Promise<void>
  onSwitchMic: (id: string) => void | Promise<void>
  onSwitchSpeaker: (id: string) => void | Promise<void>
}

function deviceLabel(d: MediaDeviceInfo, idx: number, kind: string): string {
  if (d.label) return d.label
  // Pre-permission Chrome returns blank labels. Show a positional placeholder
  // so the user can still distinguish entries.
  return `${kind} ${idx + 1}`
}

function DeviceRow({
  icon, label, value, options, kindLabel, onChange, helpText,
}: {
  icon: ReactNode
  label: string
  value: string
  options: MediaDeviceInfo[]
  kindLabel: string
  onChange: (id: string) => void
  helpText?: string
}) {
  const handleChange = (e: SelectChangeEvent<string>) => onChange(e.target.value)
  // If the current track's deviceId isn't in the enumerated list (e.g. blank
  // labels before permission), fall back to '' so the Select doesn't warn.
  const safeValue = options.some((o) => o.deviceId === value) ? value : ''

  return (
    <Box>
      <Stack direction="row" spacing={1.5} alignItems="center" sx={{ mb: 1 }}>
        <Box sx={{ color: 'text.secondary', display: 'flex' }}>{icon}</Box>
        <Typography variant="body2" sx={{ fontWeight: 600 }}>{label}</Typography>
      </Stack>
      <FormControl fullWidth size="small">
        <InputLabel id={`device-${kindLabel}`}>{kindLabel}</InputLabel>
        <Select
          labelId={`device-${kindLabel}`}
          label={kindLabel}
          value={safeValue}
          onChange={handleChange}
          disabled={options.length === 0}
          displayEmpty
        >
          {options.length === 0 && (
            <MenuItem value="" disabled>
              <em>No devices found</em>
            </MenuItem>
          )}
          {options.map((d, i) => (
            <MenuItem key={d.deviceId || `idx-${i}`} value={d.deviceId}>
              {deviceLabel(d, i, kindLabel)}
            </MenuItem>
          ))}
        </Select>
      </FormControl>
      {helpText && (
        <Typography variant="caption" color="text.secondary" sx={{ mt: 0.5, display: 'block' }}>
          {helpText}
        </Typography>
      )}
    </Box>
  )
}

export function DeviceSettingsDialog({
  open, onClose,
  cameras, mics, speakers,
  selectedCameraId, selectedMicId, selectedSpeakerId,
  onSwitchCamera, onSwitchMic, onSwitchSpeaker,
}: DeviceSettingsDialogProps) {
  // setSinkId is Chromium-only (and only over HTTPS / localhost). Detect once
  // so we can communicate the limitation in the helper line.
  const speakerSupported =
    typeof document !== 'undefined' &&
    typeof (document.createElement('video') as HTMLVideoElement & { setSinkId?: unknown }).setSinkId === 'function'

  return (
    <Dialog
      open={open}
      onClose={onClose}
      maxWidth="xs"
      fullWidth
      PaperProps={{ sx: { borderRadius: '14px', border: '1px solid rgba(255,255,255,0.06)' } }}
    >
      <DialogTitle sx={{ pb: 1.5, pt: 3, fontWeight: 600, letterSpacing: '-0.01em' }}>
        Device settings
      </DialogTitle>
      <DialogContent sx={{ pt: '8px !important' }}>
        <Stack spacing={2.5}>
          <DeviceRow
            icon={<VideocamOutlinedIcon fontSize="small" />}
            label="Camera"
            kindLabel="Camera"
            value={selectedCameraId}
            options={cameras}
            onChange={(id) => void onSwitchCamera(id)}
          />
          <DeviceRow
            icon={<MicOutlinedIcon fontSize="small" />}
            label="Microphone"
            kindLabel="Microphone"
            value={selectedMicId}
            options={mics}
            onChange={(id) => void onSwitchMic(id)}
          />
          <DeviceRow
            icon={<VolumeUpOutlinedIcon fontSize="small" />}
            label="Speaker"
            kindLabel="Speaker"
            value={selectedSpeakerId}
            options={speakers}
            onChange={(id) => void onSwitchSpeaker(id)}
            helpText={
              speakerSupported
                ? undefined
                : 'Speaker selection isn\'t supported in this browser; choose your output in your OS sound settings.'
            }
          />
        </Stack>
      </DialogContent>
      <DialogActions sx={{ px: 3, pb: 2.5, pt: 1 }}>
        <Button onClick={onClose}>Done</Button>
      </DialogActions>
    </Dialog>
  )
}
