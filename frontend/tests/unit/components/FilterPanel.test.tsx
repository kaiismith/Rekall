import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { FilterPanel } from '@/components/meetings/FilterPanel'
import type { MeetingStatusFilter } from '@/types/meeting'

function renderPanel(
  status: MeetingStatusFilter | undefined,
  onStatusChange = vi.fn(),
  onClose = vi.fn(),
) {
  // Provide a real DOM element as anchor so the Popover renders open.
  const anchor = document.createElement('div')
  document.body.appendChild(anchor)

  const result = render(
    <ThemeProvider theme={theme}>
      <FilterPanel
        anchorEl={anchor}
        onClose={onClose}
        status={status}
        onStatusChange={onStatusChange}
      />
    </ThemeProvider>,
  )

  return { ...result, onStatusChange, onClose }
}

describe('FilterPanel', () => {
  it('renders all status options', () => {
    renderPanel(undefined)
    expect(screen.getByText('All statuses')).toBeInTheDocument()
    expect(screen.getByText('in progress')).toBeInTheDocument()
    expect(screen.getByText('complete')).toBeInTheDocument()
    expect(screen.getByText('processing')).toBeInTheDocument()
    expect(screen.getByText('failed')).toBeInTheDocument()
  })

  it('calls onStatusChange with the selected value', () => {
    const { onStatusChange, onClose } = renderPanel(undefined)
    fireEvent.click(screen.getByText('complete'))
    expect(onStatusChange).toHaveBeenCalledWith('complete')
    expect(onClose).toHaveBeenCalled()
  })

  it('calls onStatusChange(undefined) when "All statuses" is selected', () => {
    const { onStatusChange } = renderPanel('complete')
    fireEvent.click(screen.getByText('All statuses'))
    expect(onStatusChange).toHaveBeenCalledWith(undefined)
  })

  it('marks the active option as selected', () => {
    renderPanel('in_progress')
    const activeItem = screen.getByRole('button', { name: 'in progress' })
    expect(activeItem).toHaveClass('Mui-selected')
  })

  it('does not mark non-active options as selected', () => {
    renderPanel('complete')
    const inProgressItem = screen.getByRole('button', { name: 'in progress' })
    expect(inProgressItem).not.toHaveClass('Mui-selected')
  })

  it('calls onStatusChange with "in_progress" for the "in progress" option', () => {
    const { onStatusChange } = renderPanel(undefined)
    fireEvent.click(screen.getByText('in progress'))
    expect(onStatusChange).toHaveBeenCalledWith('in_progress')
  })
})
