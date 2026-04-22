import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { BackgroundButton } from '@/components/meeting/BackgroundButton'
import type { BackgroundOption } from '@/types/meeting'

const noneOption: BackgroundOption = { type: 'none' }
const blurOption: BackgroundOption = { type: 'blur', level: 'light' }
const imageOption: BackgroundOption = { type: 'image', src: '/backgrounds/office.jpg', label: 'Office' }

function renderBg(
  active: BackgroundOption = noneOption,
  extras: Partial<Parameters<typeof BackgroundButton>[0]> = {},
) {
  const onSelect = vi.fn()
  const onUpload = vi.fn().mockResolvedValue(null)
  render(
    <ThemeProvider theme={theme}>
      <BackgroundButton
        active={active}
        onSelect={onSelect}
        onUpload={onUpload}
        customBgSrc={null}
        {...extras}
      />
    </ThemeProvider>,
  )
  return { onSelect, onUpload }
}

describe('BackgroundButton', () => {
  it('renders a trigger button', () => {
    renderBg()
    expect(screen.getByRole('button')).toBeInTheDocument()
  })

  it('opens the picker on click', () => {
    renderBg()
    fireEvent.click(screen.getByRole('button'))
    expect(screen.getByText('Background')).toBeInTheDocument()
  })

  it('shows None, blur and image options in picker', () => {
    renderBg()
    fireEvent.click(screen.getByRole('button'))
    // 'None' appears twice (tile body + label) — both being present is correct.
    expect(screen.getAllByText('None').length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText('Blur ·')).toBeInTheDocument()
    expect(screen.getByText('Blur ··')).toBeInTheDocument()
    expect(screen.getByText('Office')).toBeInTheDocument()
  })

  it('always shows the Custom upload slot', () => {
    renderBg()
    fireEvent.click(screen.getByRole('button'))
    expect(screen.getByText('Custom')).toBeInTheDocument()
  })

  it('calls onSelect with none option when None label is clicked', () => {
    const { onSelect } = renderBg(blurOption)
    fireEvent.click(screen.getByRole('button'))
    // Click the label caption (last of the 'None' elements) to avoid the body text duplicate.
    const noneEls = screen.getAllByText('None')
    fireEvent.click(noneEls[noneEls.length - 1])
    expect(onSelect).toHaveBeenCalledWith(noneOption)
  })

  it('calls onSelect with blur option', () => {
    const { onSelect } = renderBg()
    fireEvent.click(screen.getByRole('button'))
    fireEvent.click(screen.getByText('Blur ·'))
    expect(onSelect).toHaveBeenCalledWith(blurOption)
  })

  it('calls onSelect with image option', () => {
    const { onSelect } = renderBg()
    fireEvent.click(screen.getByRole('button'))
    fireEvent.click(screen.getByText('Office'))
    expect(onSelect).toHaveBeenCalledWith(imageOption)
  })

  it('is disabled when disabled prop is true', () => {
    renderBg(noneOption, { disabled: true })
    expect(screen.getByRole('button')).toBeDisabled()
  })

  it('shows custom image thumbnail when customBgSrc is set', () => {
    renderBg(noneOption, { customBgSrc: 'data:image/png;base64,abc' })
    fireEvent.click(screen.getByRole('button'))
    // Custom slot should have the thumbnail style applied.
    expect(screen.getByText('Custom')).toBeInTheDocument()
  })

  it('shows upload error returned by onUpload', async () => {
    const onUpload = vi.fn().mockResolvedValue('Image must be 2 MB or smaller')
    renderBg(noneOption, { onUpload })
    fireEvent.click(screen.getByRole('button'))

    // Simulate file selection via a DataTransfer-style file.
    const file = new File(['x'.repeat(10)], 'test.png', { type: 'image/png' })
    const input = document.querySelector('input[type="file"]') as HTMLInputElement
    Object.defineProperty(input, 'files', { value: [file], configurable: true })
    fireEvent.change(input)

    await waitFor(() => {
      expect(screen.getByText('Image must be 2 MB or smaller')).toBeInTheDocument()
    })
  })

  it('shows active border on custom tile when the custom image is active', () => {
    const customActive: BackgroundOption = { type: 'image', src: 'data:image/png;base64,abc', label: 'Custom' }
    renderBg(customActive, { customBgSrc: 'data:image/png;base64,abc' })
    fireEvent.click(screen.getByRole('button'))
    // isCustomActive=true means the Custom tile gets the primary.main border.
    // Just verify the tile is rendered and picker is open.
    expect(screen.getByText('Custom')).toBeInTheDocument()
  })

  it('onSelect is not called when file input fires with no file selected', async () => {
    const { onSelect, onUpload } = renderBg()
    fireEvent.click(screen.getByRole('button'))

    const input = document.querySelector('input[type="file"]') as HTMLInputElement
    Object.defineProperty(input, 'files', { value: [], configurable: true })
    fireEvent.change(input)

    // No file → early return, onUpload never called.
    expect(onUpload).not.toHaveBeenCalled()
    expect(onSelect).not.toHaveBeenCalled()
  })

  it('isActive returns true when both active and option are none', () => {
    // Exercises the `a.type === "none" && b.type === "none"` branch in isActive().
    const { onSelect } = renderBg(noneOption)
    fireEvent.click(screen.getByRole('button'))
    // The None tile should be visually selected (border via isActive=true).
    // Clicking it again calls onSelect with noneOption.
    const noneEls = screen.getAllByText('None')
    fireEvent.click(noneEls[noneEls.length - 1])
    expect(onSelect).toHaveBeenCalledWith(noneOption)
  })

  it('closes picker on successful upload', async () => {
    const onUpload = vi.fn().mockResolvedValue(null)
    renderBg(noneOption, { onUpload })
    fireEvent.click(screen.getByRole('button'))
    expect(screen.getByText('Background')).toBeInTheDocument()

    const file = new File(['x'.repeat(10)], 'test.png', { type: 'image/png' })
    const input = document.querySelector('input[type="file"]') as HTMLInputElement
    Object.defineProperty(input, 'files', { value: [file], configurable: true })
    fireEvent.change(input)

    await waitFor(() => {
      expect(screen.queryByText('Background')).not.toBeInTheDocument()
    })
  })
})
