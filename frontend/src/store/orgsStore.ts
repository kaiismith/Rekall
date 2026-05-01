import { create } from 'zustand'
import { organizationService } from '@/services/organizationService'
import type { Organization } from '@/types/organization'

interface OrgsState {
  /** null = not yet loaded; [] = loaded but empty (zero-org user). */
  orgs: Organization[] | null
  isLoading: boolean
  error: string | null

  /** Fetch the user's orgs once. Reentrant calls during an in-flight load are deduped. */
  load: () => Promise<void>
  /** Drop the cache so the next `load()` re-fetches. Call after create/delete/invitation accept. */
  invalidate: () => void
  /** Synchronous lookup used by ScopeBadge / ScopeBreadcrumb. Returns undefined while loading. */
  getOrgName: (id: string) => string | undefined
}

export const useOrgsStore = create<OrgsState>()((set, get) => ({
  orgs: null,
  isLoading: false,
  error: null,

  load: async () => {
    // Skip if a load is in flight OR if we've already attempted (orgs is
    // non-null OR error is set). This prevents the consumer-effect feedback
    // loop where `if (orgs === null) load()` re-fires hundreds of times per
    // second when /organizations responds 401 (e.g. expired session) — every
    // failed load left `orgs === null`, every render saw the gap, every
    // render re-called load.
    const { isLoading, orgs, error } = get()
    if (isLoading || orgs !== null || error !== null) return
    set({ isLoading: true, error: null })
    try {
      const fetched = await organizationService.list()
      set({ orgs: fetched, isLoading: false })
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to load organizations'
      // Set error so the load-guard above prevents retry storms; consumers
      // that want to retry must call invalidate() first (which clears error).
      set({ isLoading: false, error: msg })
    }
  },

  invalidate: () => set({ orgs: null, error: null }),

  getOrgName: (id) => get().orgs?.find((o) => o.id === id)?.name,
}))
