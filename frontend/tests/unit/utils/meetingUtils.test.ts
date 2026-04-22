import { describe, it, expect, vi, afterEach } from 'vitest'
import { formatMeetingDuration, computeElapsed, getInitials, avatarColour } from '@/utils'

// ─── formatMeetingDuration ────────────────────────────────────────────────────

describe('formatMeetingDuration', () => {
  it('returns — for zero seconds', () => {
    expect(formatMeetingDuration(0)).toBe('—')
  })

  it('returns — for negative seconds', () => {
    expect(formatMeetingDuration(-5)).toBe('—')
  })

  it('formats sub-minute durations as 0M xxS', () => {
    expect(formatMeetingDuration(45)).toBe('0M 45S')
  })

  it('formats exactly one minute', () => {
    expect(formatMeetingDuration(60)).toBe('1M 00S')
  })

  it('formats minutes and seconds', () => {
    expect(formatMeetingDuration(483)).toBe('8M 03S')
  })

  it('zero-pads seconds to two digits', () => {
    expect(formatMeetingDuration(601)).toBe('10M 01S')
  })

  it('handles large durations without hours', () => {
    expect(formatMeetingDuration(3599)).toBe('59M 59S')
  })

  it('handles 1-second duration', () => {
    expect(formatMeetingDuration(1)).toBe('0M 01S')
  })
})

// ─── computeElapsed ───────────────────────────────────────────────────────────

describe('computeElapsed', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('returns 0 for a future started_at', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2025-01-01T10:00:00Z'))
    expect(computeElapsed('2025-01-01T11:00:00Z')).toBe(0)
  })

  it('returns elapsed seconds since started_at', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2025-01-01T10:05:30Z'))
    expect(computeElapsed('2025-01-01T10:00:00Z')).toBe(330)
  })

  it('returns 0 for started_at equal to now', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2025-01-01T10:00:00Z'))
    expect(computeElapsed('2025-01-01T10:00:00Z')).toBe(0)
  })
})

// ─── getInitials ──────────────────────────────────────────────────────────────

describe('getInitials', () => {
  it('returns ? for an empty string', () => {
    expect(getInitials('')).toBe('?')
  })

  it('returns ? for whitespace-only string', () => {
    expect(getInitials('   ')).toBe('?')
  })

  it('returns single uppercase initial for a single-word name', () => {
    expect(getInitials('Alice')).toBe('A')
  })

  it('returns first and last initials for a two-word name', () => {
    expect(getInitials('Alice Smith')).toBe('AS')
  })

  it('returns first and last initials for names with multiple parts', () => {
    expect(getInitials('Mary Ann Jones')).toBe('MJ')
  })

  it('upcases initials', () => {
    expect(getInitials('bob carter')).toBe('BC')
  })

  it('handles extra spaces between words', () => {
    expect(getInitials('  John   Doe  ')).toBe('JD')
  })
})

// ─── avatarColour ─────────────────────────────────────────────────────────────

const AVATAR_PALETTE = ['#3b82f6', '#8b5cf6', '#ec4899', '#f59e0b', '#10b981', '#06b6d4']

describe('avatarColour', () => {
  it('returns a colour from the 6-colour palette', () => {
    const colour = avatarColour('user-123')
    expect(AVATAR_PALETTE).toContain(colour)
  })

  it('is deterministic — same ID always returns the same colour', () => {
    const id = 'user-abc-def'
    expect(avatarColour(id)).toBe(avatarColour(id))
  })

  it('returns different colours for different IDs across the palette', () => {
    // Generate enough IDs to exercise at least two distinct palette entries.
    const colours = new Set(
      Array.from({ length: 20 }, (_, i) => avatarColour(`user-${i}`)),
    )
    expect(colours.size).toBeGreaterThan(1)
  })

  it('handles an empty string without throwing', () => {
    expect(() => avatarColour('')).not.toThrow()
    expect(AVATAR_PALETTE).toContain(avatarColour(''))
  })
})
