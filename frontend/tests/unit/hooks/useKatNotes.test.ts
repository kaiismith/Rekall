/**
 * useKatNotes unit tests
 *
 * Verifies the bootstrap / push / dedupe / sort / cap behaviour of the
 * frontend Kat hook against a mocked katService:
 *   - Bootstrap reads /healthz/kat; configured=false flips status to offline
 *   - configured=true puts the hook in warming_up until the first note arrives
 *   - First incoming note flips warming_up -> live
 *   - Notes are sorted by window_started_at and deduped by id
 *   - Client-side cap (20) enforces FIFO eviction
 *   - retry() re-runs the probe and recovers from offline -> live on flip
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { act, renderHook, waitFor } from '@testing-library/react'

// ── Mock the katService BEFORE importing the hook ───────────────────────────

import type { KatHealthResponse, KatNoteDTO } from '@/types/kat'

const mockGetHealth = vi.fn<() => Promise<KatHealthResponse>>()

vi.mock('@/services/katService', () => ({
  katService: {
    getHealth: (): Promise<KatHealthResponse> => mockGetHealth(),
  },
}))

import { useKatNotes } from '@/hooks/useKatNotes'

const configuredHealth: KatHealthResponse = {
  configured: true,
  provider: 'foundry',
  auth_mode: 'api_key',
  deployment: 'gpt-4o-mini',
  endpoint_host: 'foundry.example.com',
}
const offlineHealth: KatHealthResponse = {
  configured: false,
  provider: '',
  auth_mode: 'none',
  deployment: '',
  endpoint_host: '',
}

function makeNote(id: string, isoStart: string): KatNoteDTO {
  return {
    id,
    run_id: 'run-' + id,
    meeting_id: 'm1',
    window_started_at: isoStart,
    window_ended_at: isoStart,
    segment_index_lo: 0,
    segment_index_hi: 0,
    summary: 'sum-' + id,
    key_points: [],
    open_questions: [],
    model_id: 'gpt-4o-mini',
    prompt_version: 'kat-v1',
  }
}

describe('useKatNotes', () => {
  beforeEach(() => {
    mockGetHealth.mockReset()
  })

  it('flips status to offline when /healthz/kat reports configured=false', async () => {
    mockGetHealth.mockResolvedValue(offlineHealth)
    const { result } = renderHook(() => useKatNotes())
    await waitFor(() => expect(result.current.status).toBe('offline'))
    expect(result.current.notes).toEqual([])
    expect(result.current.health?.configured).toBe(false)
  })

  it('stays in warming_up after a configured probe and flips to live on first note', async () => {
    mockGetHealth.mockResolvedValue(configuredHealth)
    const { result } = renderHook(() => useKatNotes())
    await waitFor(() => expect(result.current.status).toBe('warming_up'))

    act(() => {
      result.current.pushNote(makeNote('a', '2026-04-30T10:00:00Z'))
    })
    expect(result.current.status).toBe('live')
    expect(result.current.notes).toHaveLength(1)
  })

  it('sorts notes by window_started_at and deduplicates by id', async () => {
    mockGetHealth.mockResolvedValue(configuredHealth)
    const { result } = renderHook(() => useKatNotes())
    await waitFor(() => expect(result.current.status).toBe('warming_up'))

    const n1 = makeNote('a', '2026-04-30T10:00:00Z')
    const n2 = makeNote('b', '2026-04-30T10:00:20Z')
    const n3 = makeNote('c', '2026-04-30T10:00:10Z')
    const dup = makeNote('a', '2026-04-30T10:00:00Z')

    act(() => {
      result.current.pushNote(n2) // out-of-order arrival
      result.current.pushNote(n1)
      result.current.pushNote(n3)
      result.current.pushNote(dup) // duplicate id, must be dropped
    })

    expect(result.current.notes.map((n) => n.id)).toEqual(['a', 'c', 'b'])
    expect(result.current.latestNote?.id).toBe('b')
  })

  it('caps the notes array at 20 entries with FIFO eviction', async () => {
    mockGetHealth.mockResolvedValue(configuredHealth)
    const { result } = renderHook(() => useKatNotes())
    await waitFor(() => expect(result.current.status).toBe('warming_up'))

    act(() => {
      for (let i = 0; i < 25; i++) {
        // Use sortable iso timestamps so order is deterministic.
        const ts = `2026-04-30T10:${String(i).padStart(2, '0')}:00Z`
        result.current.pushNote(makeNote(`note-${String(i).padStart(2, '0')}`, ts))
      }
    })
    expect(result.current.notes).toHaveLength(20)
    // FIFO: oldest 5 dropped (note-00..note-04). Remaining ids start at note-05.
    expect(result.current.notes[0]?.id).toBe('note-05')
    expect(result.current.notes[19]?.id).toBe('note-24')
  })

  it('retry() recovers from offline to warming_up after a flip', async () => {
    mockGetHealth.mockResolvedValueOnce(offlineHealth).mockResolvedValueOnce(configuredHealth)
    const { result } = renderHook(() => useKatNotes())
    await waitFor(() => expect(result.current.status).toBe('offline'))

    await act(async () => {
      result.current.retry()
      await Promise.resolve()
    })
    await waitFor(() => expect(result.current.status).toBe('warming_up'))
  })
})
