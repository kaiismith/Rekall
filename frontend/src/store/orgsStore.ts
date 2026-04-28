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
    if (get().isLoading) return
    set({ isLoading: true, error: null })
    try {
      const orgs = await organizationService.list()
      set({ orgs, isLoading: false })
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to load organizations'
      set({ isLoading: false, error: msg })
    }
  },

  invalidate: () => set({ orgs: null, error: null }),

  getOrgName: (id) => get().orgs?.find((o) => o.id === id)?.name,
}))
