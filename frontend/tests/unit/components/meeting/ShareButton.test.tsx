import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { ShareButton } from '@/components/meeting/ShareButton'

function renderShare(props: Parameters<typeof ShareButton>[0]) {
  return render(
    <ThemeProvider theme={theme}>
      <ShareButton {...props} />
    </ThemeProvider>,
  )
}

describe('ShareButton', () => {
  it('calls onShare when not sharing and clicked', () => {
    const onShare = vi.fn()
    renderShare({ isScreenSharing: false, onShare, onStop: vi.fn() })
    fireEvent.click(screen.getByRole('button'))
    expect(onShare).toHaveBeenCalledOnce()
  })

  it('calls onStop when sharing and clicked', () => {
    const onStop = vi.fn()
    renderShare({ isScreenSharing: true, onShare: vi.fn(), onStop })
    fireEvent.click(screen.getByRole('button'))
    expect(onStop).toHaveBeenCalledOnce()
  })

  it('does not call onStop when not sharing', () => {
    const onStop = vi.fn()
    renderShare({ isScreenSharing: false, onShare: vi.fn(), onStop })
    fireEvent.click(screen.getByRole('button'))
    expect(onStop).not.toHaveBeenCalled()
  })
})
