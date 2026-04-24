import Grid from '@mui/material/Grid'
import Card from '@mui/material/Card'
import CardContent from '@mui/material/CardContent'
import Typography from '@mui/material/Typography'
import Box from '@mui/material/Box'
import Stack from '@mui/material/Stack'
import PhoneInTalkIcon from '@mui/icons-material/PhoneInTalk'
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline'
import HourglassEmptyIcon from '@mui/icons-material/HourglassEmpty'
import AccessTimeIcon from '@mui/icons-material/AccessTime'
import InsightsOutlinedIcon from '@mui/icons-material/InsightsOutlined'
import { useAuthStore } from '@/store/authStore'
import { GradientText, PageHeader } from '@/components/common/ui'

interface StatCardProps {
  label: string
  value: string | number
  delta?: string
  icon: React.ReactNode
  accent: string
}

function StatCard({ label, value, delta, icon, accent }: StatCardProps) {
  return (
    <Card
      sx={{
        position: 'relative',
        overflow: 'hidden',
        transition: 'transform 180ms ease, border-color 180ms ease, box-shadow 180ms ease',
        '&::before': {
          content: '""',
          position: 'absolute',
          top: 0,
          left: 0,
          right: 0,
          height: '1px',
          background: `linear-gradient(90deg, transparent 0%, ${accent}80 50%, transparent 100%)`,
          opacity: 0.6,
        },
        '&:hover': {
          transform: 'translateY(-2px)',
          borderColor: `${accent}4d`,
          boxShadow: `0 12px 32px -12px ${accent}40, 0 0 0 1px ${accent}1a`,
        },
      }}
    >
      <CardContent sx={{ p: 3, '&:last-child': { pb: 3 } }}>
        <Stack direction="row" justifyContent="space-between" alignItems="flex-start" spacing={2}>
          <Box sx={{ minWidth: 0 }}>
            <Typography
              variant="caption"
              color="text.secondary"
              sx={{ textTransform: 'uppercase', letterSpacing: '0.12em', fontWeight: 700, fontSize: '0.7rem' }}
            >
              {label}
            </Typography>
            <Typography
              sx={{
                fontSize: '2.125rem',
                fontWeight: 800,
                letterSpacing: '-0.03em',
                color: 'text.primary',
                mt: 0.75,
                lineHeight: 1.1,
                fontVariantNumeric: 'tabular-nums',
              }}
            >
              {value}
            </Typography>
            {delta && (
              <Typography variant="caption" sx={{ color: 'text.secondary', mt: 0.5, display: 'block' }}>
                {delta}
              </Typography>
            )}
          </Box>
          <Box
            sx={{
              width: 52,
              height: 52,
              flexShrink: 0,
              borderRadius: '14px',
              bgcolor: `${accent}1f`,
              border: `1px solid ${accent}33`,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: accent,
              boxShadow: `0 0 20px -8px ${accent}80`,
              '& .MuiSvgIcon-root': { fontSize: '1.5rem' },
            }}
          >
            {icon}
          </Box>
        </Stack>
      </CardContent>
    </Card>
  )
}

export function DashboardPage() {
  const { user } = useAuthStore()
  const greetingName = user?.full_name?.split(' ')[0] ?? 'there'

  return (
    <Box>
      <PageHeader
        documentTitleText="Dashboard"
        title={
          <>
            Welcome back, <GradientText>{greetingName}</GradientText>
          </>
        }
        subtitle="A quick look at activity across your workspace."
      />

      <Grid container spacing={2.5}>
        <Grid item xs={12} sm={6} lg={3}>
          <StatCard
            label="Total Calls"
            value="—"
            delta="No data yet"
            icon={<PhoneInTalkIcon />}
            accent="#60a5fa"
          />
        </Grid>
        <Grid item xs={12} sm={6} lg={3}>
          <StatCard
            label="Completed"
            value="—"
            delta="Awaiting first call"
            icon={<CheckCircleOutlineIcon />}
            accent="#22c55e"
          />
        </Grid>
        <Grid item xs={12} sm={6} lg={3}>
          <StatCard
            label="Processing"
            value="—"
            delta="Nothing in queue"
            icon={<HourglassEmptyIcon />}
            accent="#f59e0b"
          />
        </Grid>
        <Grid item xs={12} sm={6} lg={3}>
          <StatCard
            label="Avg Duration"
            value="—"
            delta="Across all calls"
            icon={<AccessTimeIcon />}
            accent="#a78bfa"
          />
        </Grid>
      </Grid>

      <Card sx={{ mt: 3 }}>
        <CardContent sx={{ p: { xs: 3, sm: 4 } }}>
          <Stack spacing={2} alignItems="center" textAlign="center" py={{ xs: 3, sm: 5 }}>
            <Box
              sx={{
                width: 56,
                height: 56,
                borderRadius: '14px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                bgcolor: 'rgba(129,140,248,0.1)',
                color: '#a78bfa',
              }}
            >
              <InsightsOutlinedIcon sx={{ fontSize: '1.75rem' }} />
            </Box>
            <Typography variant="h6" sx={{ fontWeight: 600, letterSpacing: '-0.01em' }}>
              Analytics arrive with your first call
            </Typography>
            <Typography variant="body2" color="text.secondary" sx={{ maxWidth: 440 }}>
              Once calls and meetings are ingested, charts for volume, duration, and
              sentiment will populate here automatically.
            </Typography>
          </Stack>
        </CardContent>
      </Card>
    </Box>
  )
}
