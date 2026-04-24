import { Button, styled, type ButtonProps } from '@mui/material'
import { forwardRef } from 'react'
import { tokens } from '@/theme'

const StyledButton = styled(Button)(({ theme }) => ({
  background: tokens.gradients.primary,
  color: '#0a0b12',
  fontWeight: 600,
  borderRadius: `${tokens.radii.button}px`,
  boxShadow: tokens.shadows.glowPrimary,
  padding: '12px 24px',
  textTransform: 'none',
  '&:hover': {
    background: tokens.gradients.primaryHover,
    boxShadow: tokens.shadows.glowPrimaryHover,
  },
  '&:focus-visible': {
    outline: `2px solid ${theme.palette.primary.light}`,
    outlineOffset: '2px',
  },
  '&.Mui-disabled': {
    background: 'rgba(255,255,255,0.05)',
    color: 'rgba(255,255,255,0.3)',
    boxShadow: 'none',
  },
}))

export const GradientButton = forwardRef<HTMLButtonElement, ButtonProps>(
  function GradientButton(props, ref) {
    return <StyledButton ref={ref} size="large" fullWidth {...props} />
  },
)
