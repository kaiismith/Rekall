import { apiClient } from './api'
import type { KatHealthResponse } from '@/types/kat'

/** Default response surfaced when `/healthz/kat` is unreachable or the
 *  backend reports Kat as unconfigured. The panel renders the offline state
 *  in either case so a 4xx/5xx never bubbles up as a toast. */
const offlineHealth: KatHealthResponse = {
  configured: false,
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
      // /healthz/kat is registered outside the /api/v1 prefix, so override
      // baseURL to hit the bare server root.
      const response = await apiClient.get<KatHealthResponse>('/healthz/kat', {
        baseURL: '/',
      })
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
