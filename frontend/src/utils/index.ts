/**
 * Format a duration in seconds to the meeting card format: "8M 00S".
 * Returns "—" for zero or negative values.
 */
export function formatMeetingDuration(seconds: number): string {
  if (seconds <= 0) return '—'
  const m = Math.floor(seconds / 60)
  const s = Math.floor(seconds % 60)
  return `${m}M ${String(s).padStart(2, '0')}S`
}

/**
 * Return the number of whole seconds elapsed since the given ISO timestamp.
 */
export function computeElapsed(startedAt: string): number {
  return Math.max(0, Math.floor((Date.now() - new Date(startedAt).getTime()) / 1000))
}

/**
 * Return up to two uppercase initials from a full name (first + last word).
 */
export function getInitials(fullName: string): string {
  // Pull only word-tokens that BEGIN with a letter — skips junk like
  // "(techvify.tc)" or "•" so the initial pair is always two letters.
  const parts = fullName
    .trim()
    .split(/\s+/)
    .filter((p) => /^[\p{L}]/u.test(p))
  if (parts.length === 0) return '?'
  const first = parts[0][0].toUpperCase()
  if (parts.length === 1) return first
  return first + parts[parts.length - 1][0].toUpperCase()
}

/**
 * Format a duration in seconds to a human-readable string (e.g. "1h 23m 45s").
 */
export function formatDuration(seconds: number): string {
  if (seconds <= 0) return '—'
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = seconds % 60
  const parts: string[] = []
  if (h > 0) parts.push(`${h}h`)
  if (m > 0) parts.push(`${m}m`)
  if (s > 0 || parts.length === 0) parts.push(`${s}s`)
  return parts.join(' ')
}

/**
 * Format an ISO date string to a localised date-time string.
 */
export function formatDateTime(iso: string): string {
  return new Intl.DateTimeFormat('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(iso))
}

/**
 * Truncate a string to the specified length, appending "…" when truncated.
 */
export function truncate(str: string, maxLength: number): string {
  if (str.length <= maxLength) return str
  return str.slice(0, maxLength - 1) + '…'
}

// Six accessible colours for participant avatars. Chosen to be distinct and
// readable against the dark-theme background.
const AVATAR_PALETTE = ['#3b82f6', '#8b5cf6', '#ec4899', '#f59e0b', '#10b981', '#06b6d4']

/**
 * Return a deterministic colour from the 6-colour avatar palette for a user ID.
 */
export function avatarColour(userId: string): string {
  let hash = 0
  for (let i = 0; i < userId.length; i++) {
    hash = userId.charCodeAt(i) + ((hash << 5) - hash)
  }
  return AVATAR_PALETTE[Math.abs(hash) % AVATAR_PALETTE.length]
}

/**
 * Return a stable derived colour string from an arbitrary string (e.g. user initials).
 */
export function stringToColor(str: string): string {
  let hash = 0
  for (let i = 0; i < str.length; i++) {
    hash = str.charCodeAt(i) + ((hash << 5) - hash)
  }
  const hue = Math.abs(hash) % 360
  return `hsl(${hue}, 60%, 55%)`
}

/**
 * Build query string from an object, omitting undefined/null values.
 */
export function buildQueryString(params: Record<string, unknown>): string {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null && value !== '') {
      search.set(
        key,
        typeof value === 'object'
          ? JSON.stringify(value)
          : String(value as string | number | boolean),
      )
    }
  }
  const qs = search.toString()
  return qs ? `?${qs}` : ''
}
