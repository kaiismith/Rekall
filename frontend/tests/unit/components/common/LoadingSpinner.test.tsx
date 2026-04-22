import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { LoadingSpinner } from '@/components/common/LoadingSpinner'

function renderSpinner(props: Parameters<typeof LoadingSpinner>[0] = {}) {
  return render(
    <ThemeProvider theme={theme}>
      <LoadingSpinner {...props} />
    </ThemeProvider>,
  )
}

describe('LoadingSpinner', () => {
  it('renders a CircularProgress element', () => {
    const { container } = renderSpinner()
    expect(container.querySelector('.MuiCircularProgress-root')).toBeInTheDocument()
  })

  it('renders inline (no minHeight 100vh) by default', () => {
    const { container } = renderSpinner()
    const wrapper = container.firstElementChild as HTMLElement
    // Default renders inside a padding Box, not a full-page Box.
    expect(wrapper?.style.minHeight ?? '').not.toBe('100vh')
  })

  it('renders a CircularProgress when fullPage=true', () => {
    const { container } = renderSpinner({ fullPage: true })
    // Both branches render the same CircularProgress — confirm it's present.
    expect(container.querySelector('.MuiCircularProgress-root')).toBeInTheDocument()
  })

  it('passes size prop to CircularProgress root element', () => {
    const { container } = renderSpinner({ size: 64 })
    const root = container.querySelector('.MuiCircularProgress-root') as HTMLElement
    // MUI applies width/height as inline style on the root span.
    expect(root?.style.width).toBe('64px')
    expect(root?.style.height).toBe('64px')
  })
})
