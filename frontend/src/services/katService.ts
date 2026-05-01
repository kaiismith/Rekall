import { apiClient } from './api'
import { API_BASE_URL } from '@/constants'
import type { KatHealthResponse } from '@/types/kat'

/** Strip the trailing `/api/v1` (or whatever API_BASE_URL points at) so the
 *  /healthz/kat probe — which is registered OUTSIDE the /api/v1 prefix on
 *  the backend — hits the bare server root. Falls back to a relative `/` if
 *  API_BASE_URL is itself a relative path. */
function buildHealthBaseURL(): string {
  if (API_BASE_URL.startsWith('http://') || API_BASE_URL.startsWith('https://')) {
    try {
      const u = new URL(API_BASE_URL)
      return `${u.protocol}//${u.host}`
    } catch {
      return '/'
    }
  }
  return '/'
}

const healthBaseURL = buildHealthBaseURL()

/** Default response surfaced when `/healthz/kat` is unreachable or the
 *  backend reports Kat as unconfigured. The panel renders the offline state
 *  in either case so a 4xx/5xx never bubbles up as a toast. */
const offlineHealth: KatHealthResponse = {
  configured: false,
  provider: '',
  auth_mode: 'none',
  deployment: '',
  endpoint_host: '',
}

/** Kat service.
 *
 *  v1 surfaces ONLY the liveness probe — Kat notes are not persisted, so
 *  there is no list endpoint to call. History is delivered over the meeting
 *  WS as part of the late-join replay (the same `kat.note` channel that
 *  carries live updates).
 */
export const katService = {
  /** Probe the backend for Kat configuration. Returns a degraded
   *  `{configured: false, ...}` response on any 4xx/5xx — never throws. */
  async getHealth(): Promise<KatHealthResponse> {
    try {
      // /healthz/kat is registered outside the /api/v1 prefix on the backend.
      // We reuse apiClient's interceptors (auth, request id, etc.) but rewrite
      // baseURL to the backend's bare host so the request doesn't fall through
      // to the frontend's SPA fallback.
      const response = await apiClient.get<KatHealthResponse>('/healthz/kat', {
        baseURL: healthBaseURL,
      })
      // Defensive: if the response body wasn't JSON (e.g. SPA fallback served
      // index.html), treat it as offline rather than corrupt state.
      if (typeof response.data !== 'object' || response.data === null) {
        console.warn('kat: getHealth returned non-JSON; treating as offline')
        return offlineHealth
      }
      return response.data
    } catch (err) {
      // Diagnostic only — Kat being offline is a first-class state, not
      // a user-facing error. The panel reads `configured: false` and
      // renders the offline card.
      console.warn('kat: getHealth failed; treating as offline', err)
      return offlineHealth
    }
  },
}
