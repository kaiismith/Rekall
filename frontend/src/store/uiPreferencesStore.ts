import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export interface UIPreferences {
  sidebarDefault: 'expanded' | 'collapsed'
  reducedMotion: boolean
  keyboardShortcutsEnabled: boolean
}

interface UIPreferencesState extends UIPreferences {
  setSidebarDefault: (v: UIPreferences['sidebarDefault']) => void
  setReducedMotion: (v: boolean) => void
  setKeyboardShortcutsEnabled: (v: boolean) => void
}

/**
 * Per-device UI preferences, persisted to localStorage under
 * `rekall_ui_prefs_v1`. Device-local by design — there's no server-side
 * store; bumping the version suffix forward-migrates the shape later.
 */
export const useUIPreferencesStore = create<UIPreferencesState>()(
  persist(
    (set) => ({
      sidebarDefault: 'expanded',
      reducedMotion: false,
      keyboardShortcutsEnabled: true,
      setSidebarDefault: (v) => set({ sidebarDefault: v }),
      setReducedMotion: (v) => set({ reducedMotion: v }),
      setKeyboardShortcutsEnabled: (v) => set({ keyboardShortcutsEnabled: v }),
    }),
    { name: 'rekall_ui_prefs_v1' },
  ),
)
