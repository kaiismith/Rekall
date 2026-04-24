import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Box, Button, Stack, Typography } from '@mui/material'
import ArrowBackIcon from '@mui/icons-material/ArrowBack'
import SearchOffOutlinedIcon from '@mui/icons-material/SearchOffOutlined'
import { ROUTES } from '@/constants'
import { ActionCard, GradientButton } from '@/components/common/ui'
import { tokens } from '@/theme'

export function NotFoundPage() {
  const navigate = useNavigate()

  useEffect(() => {
    document.title = 'Page not found — Rekall'
  }, [])

  return (
    <Box
      sx={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        px: { xs: 2, sm: 3 },
        py: { xs: 4, sm: 8 },
        position: 'relative',
        '&::before': {
          content: '""',
          position: 'absolute',
          inset: 0,
          background: tokens.gradients.heroBackground,
          pointerEvents: 'none',
        },
      }}
    >
      <ActionCard maxWidth={480} sx={{ position: 'relative' }}>
        <Stack spacing={3} alignItems="center" textAlign="center">
          <Box
            sx={{
              width: 64,
              height: 64,
              borderRadius: '16px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              bgcolor: 'rgba(129,140,248,0.1)',
              color: '#a78bfa',
            }}
          >
            <SearchOffOutlinedIcon sx={{ fontSize: '2rem' }} />
          </Box>

          <Box>
            <Typography
              sx={{
                fontSize: '0.75rem',
                fontWeight: 700,
                letterSpacing: '0.18em',
                color: 'text.secondary',
                textTransform: 'uppercase',
                mb: 1,
              }}
            >
              Error 404
            </Typography>
            <Typography
              component="h1"
              sx={{
                fontSize: { xs: '1.5rem', sm: '1.75rem' },
                fontWeight: 700,
                letterSpacing: '-0.015em',
                lineHeight: 1.2,
                color: 'text.primary',
                mb: 1,
              }}
            >
              We can&apos;t find that page
            </Typography>
            <Typography variant="body2" color="text.secondary" sx={{ maxWidth: 360, mx: 'auto' }}>
              The page you&apos;re looking for may have been moved, renamed, or no longer
              exists. Check the URL, or head back to your dashboard.
            </Typography>
          </Box>

          <Stack
            direction={{ xs: 'column', sm: 'row' }}
            spacing={1.5}
            sx={{ width: '100%', pt: 1 }}
          >
            <Button
              variant="outlined"
              fullWidth
              startIcon={<ArrowBackIcon />}
              onClick={() => navigate(-1)}
            >
              Go back
            </Button>
            <GradientButton
              fullWidth
              onClick={() => navigate(ROUTES.DASHBOARD)}
            >
              Go to dashboard
            </GradientButton>
          </Stack>
        </Stack>
      </ActionCard>
    </Box>
  )
}
