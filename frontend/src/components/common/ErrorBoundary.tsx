import { Component, type ErrorInfo, type ReactNode } from 'react'
import { Box, Button, Typography } from '@mui/material'
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline'

interface Props {
  children: ReactNode
  fallback?: ReactNode
}

interface State {
  hasError: boolean
  error: Error | null
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false, error: null }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    console.error('[ErrorBoundary]', error, info.componentStack)
  }

  handleReset = (): void => {
    this.setState({ hasError: false, error: null })
  }

  render(): ReactNode {
    if (this.state.hasError) {
      if (this.props.fallback) return this.props.fallback

      return (
        <Box
          display="flex"
          flexDirection="column"
          alignItems="center"
          justifyContent="center"
          minHeight="60vh"
          gap={3}
          px={4}
          textAlign="center"
        >
          <ErrorOutlineIcon sx={{ fontSize: 64, color: 'error.main', opacity: 0.8 }} />
          <Typography variant="h5" fontWeight={600} color="text.primary">
            Something went wrong
          </Typography>
          <Typography variant="body2" color="text.secondary" maxWidth={400}>
            {this.state.error?.message ?? 'An unexpected error occurred.'}
          </Typography>
          <Button variant="contained" onClick={this.handleReset}>
            Try again
          </Button>
        </Box>
      )
    }

    return this.props.children
  }
}
