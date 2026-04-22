import { describe, it, expect } from 'vitest'
import {
  formatDuration,
  formatDateTime,
  truncate,
  stringToColor,
  buildQueryString,
} from '@/utils'

// ─── formatDuration ───────────────────────────────────────────────────────────

describe('formatDuration', () => {
  it('returns — for zero seconds', () => {
    expect(formatDuration(0)).toBe('—')
  })

  it('returns — for negative seconds', () => {
    expect(formatDuration(-10)).toBe('—')
  })

  it('formats seconds only', () => {
    expect(formatDuration(45)).toBe('45s')
  })

  it('formats minutes and seconds', () => {
    expect(formatDuration(90)).toBe('1m 30s')
  })

  it('formats hours, minutes, and seconds', () => {
    expect(formatDuration(3661)).toBe('1h 1m 1s')
  })

  it('omits seconds when exactly on a minute boundary', () => {
    expect(formatDuration(60)).toBe('1m')
  })

  it('omits seconds when exactly on an hour boundary', () => {
    expect(formatDuration(3600)).toBe('1h')
  })

  it('omits minutes when zero', () => {
    expect(formatDuration(3601)).toBe('1h 1s')
  })

  it('handles exactly one hour', () => {
    expect(formatDuration(3600)).toBe('1h')
  })

  it('handles a realistic 30-minute call', () => {
    expect(formatDuration(1800)).toBe('30m')
  })
})

// ─── formatDateTime ───────────────────────────────────────────────────────────

describe('formatDateTime', () => {
  it('returns a non-empty string for a valid ISO date', () => {
    const result = formatDateTime('2024-01-15T10:30:00Z')
    expect(result).toBeTruthy()
    expect(typeof result).toBe('string')
  })

  it('includes the year in the output', () => {
    const result = formatDateTime('2024-06-01T00:00:00Z')
    expect(result).toContain('2024')
  })

  it('includes the day of the month', () => {
    const result = formatDateTime('2024-01-15T00:00:00Z')
    expect(result).toContain('15')
  })

  it('produces different output for different dates', () => {
    const a = formatDateTime('2024-01-01T00:00:00Z')
    const b = formatDateTime('2024-12-31T00:00:00Z')
    expect(a).not.toBe(b)
  })
})

// ─── truncate ─────────────────────────────────────────────────────────────────

describe('truncate', () => {
  it('returns the string unchanged when shorter than maxLength', () => {
    expect(truncate('hello', 10)).toBe('hello')
  })

  it('returns the string unchanged when exactly maxLength', () => {
    expect(truncate('hello', 5)).toBe('hello')
  })

  it('truncates and appends ellipsis when longer than maxLength', () => {
    expect(truncate('hello world', 8)).toBe('hello w…')
  })

  it('result length equals maxLength after truncation', () => {
    const result = truncate('abcdefghij', 6)
    expect(result).toHaveLength(6)
    expect(result.endsWith('…')).toBe(true)
  })

  it('handles maxLength of 1', () => {
    expect(truncate('abc', 1)).toBe('…')
  })

  it('handles empty string input', () => {
    expect(truncate('', 5)).toBe('')
  })

  it('handles strings with unicode characters', () => {
    const result = truncate('hello 🌍 world', 8)
    expect(result.endsWith('…')).toBe(true)
    expect(result).toHaveLength(8)
  })
})

// ─── stringToColor ────────────────────────────────────────────────────────────

describe('stringToColor', () => {
  it('returns a CSS hsl string', () => {
    const result = stringToColor('alice')
    expect(result).toMatch(/^hsl\(\d+, 60%, 55%\)$/)
  })

  it('is deterministic — same input always produces same output', () => {
    expect(stringToColor('bob')).toBe(stringToColor('bob'))
    expect(stringToColor('alice@example.com')).toBe(stringToColor('alice@example.com'))
  })

  it('produces different colours for different inputs', () => {
    expect(stringToColor('alice')).not.toBe(stringToColor('bob'))
  })

  it('handles an empty string without throwing', () => {
    expect(() => stringToColor('')).not.toThrow()
  })

  it('hue is in valid range 0–359', () => {
    const match = stringToColor('test').match(/^hsl\((\d+),/)
    expect(match).not.toBeNull()
    const hue = parseInt(match![1], 10)
    expect(hue).toBeGreaterThanOrEqual(0)
    expect(hue).toBeLessThan(360)
  })
})

// ─── buildQueryString ─────────────────────────────────────────────────────────

describe('buildQueryString', () => {
  it('returns empty string for empty params', () => {
    expect(buildQueryString({})).toBe('')
  })

  it('builds a query string from simple key-value pairs', () => {
    expect(buildQueryString({ page: 1, per_page: 20 })).toBe('?page=1&per_page=20')
  })

  it('omits keys with undefined values', () => {
    const result = buildQueryString({ page: 1, status: undefined })
    expect(result).toBe('?page=1')
    expect(result).not.toContain('status')
  })

  it('omits keys with null values', () => {
    const result = buildQueryString({ page: 1, user_id: null })
    expect(result).toBe('?page=1')
    expect(result).not.toContain('user_id')
  })

  it('omits keys with empty string values', () => {
    const result = buildQueryString({ page: 1, filter: '' })
    expect(result).toBe('?page=1')
    expect(result).not.toContain('filter')
  })

  it('includes boolean values', () => {
    const result = buildQueryString({ active: true })
    expect(result).toBe('?active=true')
  })

  it('serialises object values as JSON', () => {
    const result = buildQueryString({ meta: { key: 'val' } })
    expect(result).toContain('meta=')
    expect(decodeURIComponent(result)).toContain('{"key":"val"}')
  })

  it('includes string values unchanged', () => {
    const result = buildQueryString({ status: 'pending' })
    expect(result).toBe('?status=pending')
  })

  it('handles all-undefined params by returning empty string', () => {
    expect(buildQueryString({ a: undefined, b: null, c: '' })).toBe('')
  })
})
