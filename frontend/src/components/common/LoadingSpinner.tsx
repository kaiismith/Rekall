import { Box, CircularProgress, type CircularProgressProps } from '@mui/material'

interface LoadingSpinnerProps {
  size?: number
  color?: CircularProgressProps['color']
  fullPage?: boolean
}

export function LoadingSpinner({ size = 40, color = 'primary', fullPage = false }: LoadingSpinnerProps) {
  if (fullPage) {
    return (
      <Box
        display="flex"
        alignItems="center"
        justifyContent="center"
        minHeight="100vh"
        bgcolor="background.default"
      >
        <CircularProgress size={size} color={color} />
      </Box>
    )
  }

  return (
    <Box display="flex" alignItems="center" justifyContent="center" p={4}>
      <CircularProgress size={size} color={color} />
    </Box>
  )
}
