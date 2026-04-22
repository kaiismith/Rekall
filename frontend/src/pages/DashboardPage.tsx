import Grid from '@mui/material/Grid'
import Card from '@mui/material/Card'
import CardContent from '@mui/material/CardContent'
import Typography from '@mui/material/Typography'
import Box from '@mui/material/Box'
import PhoneInTalkIcon from '@mui/icons-material/PhoneInTalk'
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline'
import HourglassEmptyIcon from '@mui/icons-material/HourglassEmpty'
import AccessTimeIcon from '@mui/icons-material/AccessTime'
import { PageTitle } from '@/components/common/PageTitle'

interface StatCardProps {
  label: string
  value: string | number
  icon: React.ReactNode
  iconColor: string
}

function StatCard({ label, value, icon, iconColor }: StatCardProps) {
  return (
    <Card>
      <CardContent sx={{ p: 2.5, '&:last-child': { pb: 2.5 } }}>
        <Box display="flex" alignItems="center" justifyContent="space-between">
          <Box>
            <Typography variant="caption" color="text.secondary" textTransform="uppercase" letterSpacing="0.05em">
              {label}
            </Typography>
            <Typography variant="h4" fontWeight={700} color="text.primary" mt={0.5}>
              {value}
            </Typography>
          </Box>
          <Box
            sx={{
              width: 48,
              height: 48,
              borderRadius: '10px',
              bgcolor: `${iconColor}22`,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: iconColor,
            }}
          >
            {icon}
          </Box>
        </Box>
      </CardContent>
    </Card>
  )
}

/**
 * Dashboard overview — placeholder stats until data is wired.
 */
export function DashboardPage() {
  return (
    <>
      <PageTitle title="Dashboard" subtitle="Welcome to Rekall Call Intelligence Platform" />

      <Grid container spacing={2.5}>
        <Grid item xs={12} sm={6} md={3}>
          <StatCard
            label="Total Calls"
            value="—"
            icon={<PhoneInTalkIcon />}
            iconColor="#3b82f6"
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <StatCard
            label="Completed"
            value="—"
            icon={<CheckCircleOutlineIcon />}
            iconColor="#22c55e"
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <StatCard
            label="Processing"
            value="—"
            icon={<HourglassEmptyIcon />}
            iconColor="#f59e0b"
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <StatCard
            label="Avg Duration"
            value="—"
            icon={<AccessTimeIcon />}
            iconColor="#8b5cf6"
          />
        </Grid>
      </Grid>

      <Box mt={4}>
        <Card>
          <CardContent sx={{ p: 3 }}>
            <Typography variant="body2" color="text.secondary" textAlign="center" py={4}>
              Analytics charts will appear here once call data is available.
            </Typography>
          </CardContent>
        </Card>
      </Box>
    </>
  )
}
