import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Typography from '@mui/material/Typography'
import { useNavigate } from 'react-router-dom'
import { ROUTES } from '@/constants'
import { PageTitle } from '@/components/common/PageTitle'

export function NotFoundPage() {
  const navigate = useNavigate()

  return (
    <Box
      display="flex"
      flexDirection="column"
      alignItems="center"
      justifyContent="center"
      minHeight="60vh"
      textAlign="center"
      gap={2}
    >
      <PageTitle title="404" documentTitle={true} />
      <Typography
        sx={{ fontSize: '6rem', fontWeight: 700, color: 'primary.main', lineHeight: 1 }}
      >
        404
      </Typography>
      <Typography variant="h5" fontWeight={600} color="text.primary">
        Page not found
      </Typography>
      <Typography variant="body2" color="text.secondary" maxWidth={360}>
        The page you&apos;re looking for doesn&apos;t exist or has been moved.
      </Typography>
      <Button variant="contained" onClick={() => navigate(ROUTES.DASHBOARD)} sx={{ mt: 1 }}>
        Back to Dashboard
      </Button>
    </Box>
  )
}
