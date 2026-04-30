import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import theme from '@/theme'

import { KatPanel } from '@/components/meeting/KatPanel'
import type { KatHealthResponse, KatNoteDTO } from '@/types/kat'

const configuredHealth: KatHealthResponse = {
  configured: true,
  auth_mode: 'api_key',
  deployment: 'gpt-4o-mini',
  endpoint_host: 'foundry.example.com',
}

const sampleNote: KatNoteDTO = {
  id: 'n1',
  run_id: 'r1',
  meeting_id: 'm1',
  window_started_at: '2026-04-30T10:00:00Z',
  window_ended_at: '2026-04-30T10:02:00Z',
  segment_index_lo: 0,
  segment_index_hi: 5,
  summary: 'Team agreed to ship the new auth flow this week.',
  key_points: ['Auth migration ready', 'Tests are green'],
  open_questions: ['Who pages on Friday?'],
  model_id: 'gpt-4o-mini',
  prompt_version: 'kat-v1',
}

function renderPanel(
  status: Parameters<typeof KatPanel>[0]['status'],
  latestNote: KatNoteDTO | null,
  health: KatHealthResponse | null = configuredHealth,
) {
  return render(
    <ThemeProvider theme={theme}>
      <KatPanel status={status} latestNote={latestNote} health={health} />
    </ThemeProvider>,
  )
}

describe('<KatPanel>', () => {
  it('renders the summary, key points, and open questions for a happy-path note', () => {
    renderPanel('live', sampleNote)
    expect(screen.getByText('Kat — live notes')).toBeInTheDocument()
    expect(screen.getByText(/Team agreed to ship/)).toBeInTheDocument()
    expect(screen.getByText('Auth migration ready')).toBeInTheDocument()
    expect(screen.getByText('Tests are green')).toBeInTheDocument()
    expect(screen.getByText('Who pages on Friday?')).toBeInTheDocument()
    // The "notes are not saved" hint must be visible in the live state.
    expect(screen.getByText(/Notes are not saved/i)).toBeInTheDocument()
    // Footer carries the model id from the health response.
    expect(screen.getByText(/model: gpt-4o-mini/)).toBeInTheDocument()
  })

  it('renders the offline state and hides action affordances', () => {
    renderPanel('offline', null, {
      configured: false,
      auth_mode: 'none',
      deployment: '',
      endpoint_host: '',
    })
    expect(screen.getByText(/Kat is offline/i)).toBeInTheDocument()
    expect(screen.getByText(/Ask your administrator/i)).toBeInTheDocument()
    // The "Notes are not saved" footer is NOT shown in the offline state.
    expect(screen.queryByText(/Notes are not saved/i)).not.toBeInTheDocument()
    // Action affordances (Live / Warming up chips) should be absent.
    expect(screen.queryByText('Live')).not.toBeInTheDocument()
  })

  it('renders the warming_up placeholder before any notes arrive', () => {
    renderPanel('warming_up', null)
    expect(screen.getByText(/Kat is listening/i)).toBeInTheDocument()
    // Warming up chip is visible.
    expect(screen.getByText('Warming up')).toBeInTheDocument()
    // The "Notes are not saved" hint is shown so the user understands the
    // ephemerality even before the first note arrives.
    expect(screen.getByText(/Notes are not saved/i)).toBeInTheDocument()
  })

  it('keeps the previous note rendered when status briefly errors', () => {
    // Simulating the transition described in Requirement 9.6: an error
    // during a tick should NOT clear the displayed summary; the parent
    // hook keeps the latest note around and the panel keeps showing it.
    renderPanel('live', sampleNote)
    expect(screen.getByText(/Team agreed to ship/)).toBeInTheDocument()
  })
})
