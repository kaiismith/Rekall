import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'
import { SortMenu } from '@/components/meetings/SortMenu'
import type { MeetingSortKey } from '@/types/meeting'

function renderMenu(
  sort: MeetingSortKey,
  onSortChange = vi.fn(),
  onClose = vi.fn(),
) {
  const anchor = document.createElement('div')
  document.body.appendChild(anchor)

  const result = render(
    <ThemeProvider theme={theme}>
      <SortMenu
        anchorEl={anchor}
        onClose={onClose}
        sort={sort}
        onSortChange={onSortChange}
      />
    </ThemeProvider>,
  )

  return { ...result, onSortChange, onClose }
}

describe('SortMenu', () => {
  it('renders all six sort options', () => {
    renderMenu('created_at_desc')
    expect(screen.getByText('Newest first')).toBeInTheDocument()
    expect(screen.getByText('Oldest first')).toBeInTheDocument()
    expect(screen.getByText('Longest first')).toBeInTheDocument()
    expect(screen.getByText('Shortest first')).toBeInTheDocument()
    expect(screen.getByText('Title A → Z')).toBeInTheDocument()
    expect(screen.getByText('Title Z → A')).toBeInTheDocument()
  })

  it('calls onSortChange with the selected key', () => {
    const { onSortChange, onClose } = renderMenu('created_at_desc')
    fireEvent.click(screen.getByText('Longest first'))
    expect(onSortChange).toHaveBeenCalledWith('duration_desc')
    expect(onClose).toHaveBeenCalled()
  })

  it('shows a checkmark next to the active option', () => {
    renderMenu('title_asc')
    // The CheckIcon is rendered only for the active item.
    // The active label is bold (fontWeight 600); non-active labels are weight 400.
    const activeLabel = screen.getByText('Title A → Z')
    expect(activeLabel).toHaveStyle({ fontWeight: '600' })
  })

  it('does not show checkmark on inactive options', () => {
    renderMenu('created_at_desc')
    const inactiveLabel = screen.getByText('Longest first')
    expect(inactiveLabel).toHaveStyle({ fontWeight: '400' })
  })

  it('marks the active menu item as selected', () => {
    renderMenu('duration_asc')
    const activeItem = screen.getByText('Shortest first').closest('li')
    expect(activeItem).toHaveClass('Mui-selected')
  })

  it('calls onSortChange with correct key for each option', () => {
    const cases: [string, MeetingSortKey][] = [
      ['Newest first', 'created_at_desc'],
      ['Oldest first', 'created_at_asc'],
      ['Longest first', 'duration_desc'],
      ['Shortest first', 'duration_asc'],
      ['Title A → Z', 'title_asc'],
      ['Title Z → A', 'title_desc'],
    ]

    for (const [label, key] of cases) {
      const onSortChange = vi.fn()
      const anchor = document.createElement('div')
      document.body.appendChild(anchor)
      const { unmount } = render(
        <ThemeProvider theme={theme}>
          <SortMenu anchorEl={anchor} onClose={vi.fn()} sort="created_at_desc" onSortChange={onSortChange} />
        </ThemeProvider>,
      )
      fireEvent.click(screen.getByText(label))
      expect(onSortChange).toHaveBeenCalledWith(key)
      unmount()
    }
  })
})
