import { useRef, useState } from 'react'
import { Alert, Box, IconButton, Paper, Popover, Tooltip, Typography } from '@mui/material'
import BlurOnIcon from '@mui/icons-material/BlurOn'
import AddPhotoAlternateIcon from '@mui/icons-material/AddPhotoAlternate'
import { BACKGROUND_OPTIONS } from '@/config/meetingControls'
import type { BackgroundOption } from '@/types/meeting'

interface BackgroundButtonProps {
  active: BackgroundOption
  onSelect: (option: BackgroundOption) => void
  onUpload: (file: File) => Promise<string | null>
  customBgSrc?: string | null
  disabled?: boolean
}

function optionLabel(opt: BackgroundOption): string {
  if (opt.type === 'none') return 'None'
  if (opt.type === 'blur') return opt.level === 'light' ? 'Blur ·' : 'Blur ··'
  return opt.label
}

function optionKey(opt: BackgroundOption): string {
  if (opt.type === 'none') return 'none'
  if (opt.type === 'blur') return `blur-${opt.level}`
  return `image-${opt.src}`
}

function isActive(a: BackgroundOption, b: BackgroundOption): boolean {
  if (a.type !== b.type) return false
  if (a.type === 'blur' && b.type === 'blur') return a.level === b.level
  if (a.type === 'image' && b.type === 'image') return a.src === b.src
  return a.type === 'none' && b.type === 'none'
}

export function BackgroundButton({ active, onSelect, onUpload, customBgSrc, disabled }: BackgroundButtonProps) {
  const [open, setOpen] = useState(false)
  const [uploadError, setUploadError] = useState<string | null>(null)
  const [uploading, setUploading] = useState(false)
  const anchorRef = useRef<HTMLButtonElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    // Reset so the same file can be re-selected if needed.
    e.target.value = ''
    if (!file) return
    setUploadError(null)
    setUploading(true)
    const error = await onUpload(file)
    setUploading(false)
    if (error) {
      setUploadError(error)
    } else {
      setOpen(false)
    }
  }

  const isCustomActive = active.type === 'image' && active.label === 'Custom'

  return (
    <>
      <Tooltip title={disabled ? 'Background effects not supported in this browser' : 'Change background'}>
        <span>
          <IconButton
            ref={anchorRef}
            onClick={() => !disabled && setOpen((v) => !v)}
            size="medium"
            disabled={disabled}
            sx={{
              bgcolor: active.type !== 'none' ? 'secondary.main' : 'action.selected',
              color: 'white',
              '&:hover': { bgcolor: active.type !== 'none' ? 'secondary.dark' : 'action.hover' },
            }}
          >
            <BlurOnIcon />
          </IconButton>
        </span>
      </Tooltip>

      {/* Hidden file input — accept images only, no size attr (we validate in hook) */}
      <input
        ref={fileInputRef}
        type="file"
        accept="image/*"
        style={{ display: 'none' }}
        onChange={handleFileChange}
      />

      <Popover
        open={open}
        anchorEl={anchorRef.current}
        onClose={() => { setOpen(false); setUploadError(null) }}
        anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
        transformOrigin={{ vertical: 'bottom', horizontal: 'center' }}
        slotProps={{ paper: { sx: { bgcolor: 'background.paper', borderRadius: 2 } } }}
      >
        <Paper elevation={4} sx={{ p: 1.5 }}>
          <Typography variant="caption" sx={{ color: 'text.secondary', display: 'block', mb: 1 }}>
            Background
          </Typography>

          {uploadError && (
            <Alert severity="error" onClose={() => setUploadError(null)} sx={{ mb: 1, py: 0, fontSize: '0.7rem' }}>
              {uploadError}
            </Alert>
          )}

          <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(4, 72px)', gap: 1 }}>
            {/* Preset options */}
            {BACKGROUND_OPTIONS.map((opt) => {
              const key = optionKey(opt)
              const label = optionLabel(opt)
              const selected = isActive(opt, active)

              return (
                <Box
                  key={key}
                  onClick={() => { onSelect(opt); setOpen(false) }}
                  sx={{
                    width: 72,
                    cursor: 'pointer',
                    borderRadius: 1,
                    overflow: 'hidden',
                    border: '2px solid',
                    borderColor: selected ? 'primary.main' : 'transparent',
                    '&:hover': { borderColor: 'primary.light' },
                  }}
                >
                  <Box
                    sx={{
                      width: '100%',
                      aspectRatio: '16/9',
                      bgcolor: 'background.default',
                      backgroundImage: opt.type === 'image' ? `url(${opt.src})` : undefined,
                      backgroundSize: 'cover',
                      backgroundPosition: 'center',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      filter: opt.type === 'blur'
                        ? opt.level === 'light' ? 'blur(3px)' : 'blur(6px)'
                        : undefined,
                    }}
                  >
                    {opt.type === 'none' && (
                      <Typography variant="caption" sx={{ color: 'text.secondary' }}>None</Typography>
                    )}
                  </Box>
                  <Typography variant="caption" sx={{ display: 'block', textAlign: 'center', py: 0.25, fontSize: '0.65rem' }}>
                    {label}
                  </Typography>
                </Box>
              )
            })}

            {/* Custom image slot — one slot, always last */}
            <Tooltip title={customBgSrc ? 'Replace custom image (max 2 MB)' : 'Upload custom image (max 2 MB)'}>
              <Box
                onClick={() => !uploading && fileInputRef.current?.click()}
                sx={{
                  width: 72,
                  cursor: uploading ? 'wait' : 'pointer',
                  borderRadius: 1,
                  overflow: 'hidden',
                  border: '2px solid',
                  borderColor: isCustomActive ? 'primary.main' : 'transparent',
                  '&:hover': { borderColor: 'primary.light' },
                  position: 'relative',
                }}
              >
                <Box
                  sx={{
                    width: '100%',
                    aspectRatio: '16/9',
                    bgcolor: 'background.default',
                    backgroundImage: customBgSrc ? `url(${customBgSrc})` : undefined,
                    backgroundSize: 'cover',
                    backgroundPosition: 'center',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                  }}
                >
                  {!customBgSrc && (
                    <AddPhotoAlternateIcon sx={{ fontSize: '1.1rem', color: 'text.disabled' }} />
                  )}
                </Box>
                <Typography variant="caption" sx={{ display: 'block', textAlign: 'center', py: 0.25, fontSize: '0.65rem' }}>
                  {uploading ? '…' : 'Custom'}
                </Typography>
              </Box>
            </Tooltip>
          </Box>
        </Paper>
      </Popover>
    </>
  )
}
