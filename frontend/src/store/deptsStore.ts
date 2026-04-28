import { create } from 'zustand'
import { organizationService } from '@/services/organizationService'
import type { Department } from '@/types/organization'

interface DeptsState {
  /** byOrg[orgId] = list of departments in that org; undefined = not yet loaded. */
  byOrg: Record<string, Department[] | undefined>
  isLoading: Record<string, boolean>
  errors: Record<string, string | undefined>

  /** Idempotent: returns immediately if the org's depts are already cached or in flight. */
  ensureLoaded: (orgId: string) => Promise<void>
  /** Drop the cache for one org. Call after create/delete dept. */
  invalidate: (orgId: string) => void
  /** Synchronous name lookup; undefined while loading. */
  getDeptName: (orgId: string, deptId: string) => string | undefined
  /** Synchronous full-list accessor; undefined while loading. */
  listForOrg: (orgId: string) => Department[] | undefined
}

export const useDeptsStore = create<DeptsState>()((set, get) => ({
  byOrg: {},
  isLoading: {},
  errors: {},

  ensureLoaded: async (orgId) => {
    const state = get()
    if (state.byOrg[orgId] !== undefined) return
    if (state.isLoading[orgId]) return
    set((s) => ({
      isLoading: { ...s.isLoading, [orgId]: true },
      errors: { ...s.errors, [orgId]: undefined },
    }))
    try {
      const depts = await organizationService.listDepartments(orgId)
      set((s) => ({
        byOrg: { ...s.byOrg, [orgId]: depts },
        isLoading: { ...s.isLoading, [orgId]: false },
      }))
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to load departments'
      set((s) => ({
        isLoading: { ...s.isLoading, [orgId]: false },
        errors: { ...s.errors, [orgId]: msg },
      }))
    }
  },

  invalidate: (orgId) =>
    set((s) => {
      const next: Record<string, Department[] | undefined> = { ...s.byOrg }
      delete next[orgId]
      return { byOrg: next }
    }),

  getDeptName: (orgId, deptId) => get().byOrg[orgId]?.find((d) => d.id === deptId)?.name,

  listForOrg: (orgId) => get().byOrg[orgId],
}))
