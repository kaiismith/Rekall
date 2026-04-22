import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Box,
  Button,
  FormControl,
  FormHelperText,
  InputLabel,
  MenuItem,
  Paper,
  Select,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import { meetingService } from '@/services/meetingService'
import type { MeetingType } from '@/types/meeting'

export function NewMeetingPage() {
  const navigate = useNavigate()
  const [title, setTitle] = useState('')
  const [type, setType] = useState<MeetingType>('open')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const handleCreate = async () => {
    setError(null)
    setLoading(true)
    try {
      const res = await meetingService.create({ title, type })
      navigate(`/meeting/${res.data.code}`)
    } catch (e: unknown) {
      const msg = (e as { response?: { data?: { error?: { message?: string } } } })
        ?.response?.data?.error?.message
      setError(msg ?? 'Failed to create meeting.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Box sx={{ p: 3, maxWidth: 480, mx: 'auto' }}>
      <Typography variant="h5" fontWeight={700} mb={3}>
        New Meeting
      </Typography>
      <Paper sx={{ p: 3 }}>
        <Stack spacing={3}>
          <TextField
            label="Title (optional)"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            fullWidth
            placeholder="e.g. Weekly Team Sync"
          />

          <FormControl fullWidth>
            <InputLabel>Meeting Type</InputLabel>
            <Select
              value={type}
              label="Meeting Type"
              onChange={(e) => setType(e.target.value as MeetingType)}
            >
              <MenuItem value="open">
                Open — anyone authenticated can join
              </MenuItem>
              <MenuItem value="private">
                Private — org/dept members only (others must knock)
              </MenuItem>
            </Select>
            {type === 'private' && (
              <FormHelperText>
                After creating, configure the scope in meeting settings.
              </FormHelperText>
            )}
          </FormControl>

          {error && (
            <Typography color="error" variant="body2">
              {error}
            </Typography>
          )}

          <Stack direction="row" spacing={1} justifyContent="flex-end">
            <Button variant="outlined" onClick={() => navigate(-1)} disabled={loading}>
              Cancel
            </Button>
            <Button variant="contained" onClick={handleCreate} disabled={loading}>
              {loading ? 'Creating…' : 'Create & Join'}
            </Button>
          </Stack>
        </Stack>
      </Paper>
    </Box>
  )
}
