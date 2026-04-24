/**
 * Tuning constants for the meeting chat feature. Collected here so product
 * can adjust limits (rate limit, grouping, page size) without hunting through
 * component files.
 */

/** Maximum body length in characters. Enforced client-side; the server also
 *  validates (see backend `maxChatMessageLength`). */
export const MAX_MESSAGE_LENGTH = 2000

/** Character count at which the counter turns red to warn the user. */
export const CHAR_COUNTER_WARN_AT = 1900

/** Sliding rate-limit window. No more than RATE_LIMIT_MESSAGES sends within
 *  RATE_LIMIT_WINDOW_MS. */
export const RATE_LIMIT_WINDOW_MS = 2000
export const RATE_LIMIT_MESSAGES = 3

/** Messages from the same sender within this window are visually grouped
 *  (no repeated avatar/name). */
export const GROUPING_WINDOW_MS = 3 * 60_000

/** Default and maximum history page sizes. Must match the backend clamp. */
export const HISTORY_PAGE_SIZE = 50
export const HISTORY_PAGE_SIZE_MAX = 100

/** If the list is scrolled within this many px of the bottom, incoming
 *  messages auto-scroll. Otherwise the "↓ New messages" pill is shown. */
export const AUTO_SCROLL_THRESHOLD_PX = 64

/** Desktop drawer width. */
export const CHAT_PANEL_WIDTH_PX = 360

/** Mark a pending optimistic message as failed if the server echo does not
 *  arrive within this window. The user can then retry. */
export const PENDING_MESSAGE_TIMEOUT_MS = 10_000
