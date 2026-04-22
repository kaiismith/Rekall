import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { CameraButton } from '@/components/meeting/CameraButton'

function renderCamera(props: Parameters<typeof CameraButton>[0]) {
  return render(
    <ThemeProvider theme={theme}>
      <CameraButton {...props} />
    </ThemeProvider>,
  )
}

describe('CameraButton', () => {
  it('renders a button', () => {
    renderCamera({ isCameraOff: false, onToggle: vi.fn() })
    expect(screen.getByRole('button')).toBeInTheDocument()
  })

  it('calls onToggle when clicked', () => {
    const onToggle = vi.fn()
    renderCamera({ isCameraOff: false, onToggle })
    fireEvent.click(screen.getByRole('button'))
    expect(onToggle).toHaveBeenCalledOnce()
  })

  it('calls onToggle when camera is off and clicked', () => {
    const onToggle = vi.fn()
    renderCamera({ isCameraOff: true, onToggle })
    fireEvent.click(screen.getByRole('button'))
    expect(onToggle).toHaveBeenCalledOnce()
  })
})
