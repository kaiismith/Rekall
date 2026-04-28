import { useCallback } from 'react'
import { ApiError } from '@/services/api'

/**
 * Returned helper. Pass any caught error from a mutation; if it's a 403,
 * the helper:
 *   1. Surfaces the inline "no longer have permission" message via the
 *      `notify` callback supplied by the caller (page-level Alert / Snackbar).
 *   2. Invokes the `invalidate` callback so the caller's local state (e.g.
 *      `orgsStore.invalidate()`) reflects current server reality on the
 *      next render.
 *
 * Returns `true` when the error was handled here and the caller can stop
 * processing it; `false` otherwise so the caller's existing error path can
 * still surface the message.
 */
export type StalePermissionHandler = (err: unknown) => boolean

interface Options {
  /**
   * Drop the relevant client-side cache so a fresh fetch reflects the new
   * permission state. For org-membership-driven 403s this is typically
   * `() => orgsStore.invalidate()`.
   */
  invalidate?: () => void
  /**
   * Surface a user-facing message. Caller is responsible for the actual UI —
   * the hook stays decoupled from any specific snackbar/toast library.
   */
  notify?: (message: string) => void
}

const DEFAULT_MESSAGE =
  'You no longer have permission to perform this action. Refresh to see the latest state.'

export function useStalePermissionHandler(opts: Options = {}): StalePermissionHandler {
  const { invalidate, notify } = opts

  return useCallback(
    (err: unknown) => {
      const status =
        err instanceof ApiError ? err.status : (err as { response?: { status?: number } })?.response?.status
      if (status !== 403) return false
      notify?.(DEFAULT_MESSAGE)
      invalidate?.()
      return true
    },
    [invalidate, notify],
  )
}
