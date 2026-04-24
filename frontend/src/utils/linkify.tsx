import type { ReactNode } from 'react'

/**
 * Matches http(s) URLs. The trailing-character exclusion list (`.,)]}"'`) keeps
 * punctuation that commonly follows URLs out of the match — e.g. "see
 * https://example.com." yields a link without the period.
 *
 * The regex is compiled once at module load and reused via `lastIndex` in the
 * splitter, so rendering N messages does not recompile the pattern.
 */
const URL_RE = /\bhttps?:\/\/[^\s<]+[^\s<.,)\]}"']/gi

/**
 * Split `body` into plain-text segments and anchor elements wrapping detected
 * URLs. All non-URL content is returned as strings; React's default rendering
 * escapes them, so arbitrary HTML / script input cannot reach the DOM.
 *
 * Example:
 *   linkify('visit https://example.com now')
 *   → ['visit ', <a href="https://example.com">…</a>, ' now']
 */
export function linkify(body: string): ReactNode[] {
  const out: ReactNode[] = []
  URL_RE.lastIndex = 0
  let last = 0
  let match: RegExpExecArray | null
  while ((match = URL_RE.exec(body)) !== null) {
    const start = match.index
    const end = start + match[0].length
    if (start > last) out.push(body.slice(last, start))
    out.push(
      <a
        key={`${start}-${match[0]}`}
        href={match[0]}
        target="_blank"
        rel="noopener noreferrer"
      >
        {match[0]}
      </a>,
    )
    last = end
  }
  if (last < body.length) out.push(body.slice(last))
  return out
}
