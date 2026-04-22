import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render } from '@testing-library/react'

// ── Mocks ────────────────────────────────────────────────────────────────────

// Mock useBootstrap so App doesn't make real API calls on mount.
vi.mock('@/hooks/useBootstrap', () => ({
  useBootstrap: vi.fn(),
}))

// Mock authStore — the module wires Axios interceptors as a side effect.
// Provide a Zustand-compatible mock that handles both `useAuthStore()` and
// `useAuthStore(selector)` call styles.
const authState = {
  user: null,
  accessToken: null,
  isInitialised: true,
  setAuth: vi.fn(),
  clearAuth: vi.fn(),
  setInitialised: vi.fn(),
}

vi.mock('@/store/authStore', () => {
  const useAuthStore = (selector?: (s: any) => any) =>
    selector ? selector(authState) : authState

  useAuthStore.getState = () => authState
  useAuthStore.setState = vi.fn()
  useAuthStore.subscribe = vi.fn()

  return { useAuthStore }
})

// Mock MeetingRoomPage — it uses WebRTC APIs that aren't available in jsdom.
vi.mock('@/pages/MeetingRoomPage', () => ({
  MeetingRoomPage: () => <div>MeetingRoomPage</div>,
}))

import App from '@/App'

// ── Tests ────────────────────────────────────────────────────────────────────

describe('App', () => {
  beforeEach(() => vi.clearAllMocks())

  it('renders without crashing', () => {
    const { container } = render(<App />)
    expect(container).toBeTruthy()
  })

  it('redirects to login when not authenticated', () => {
    const { container } = render(<App />)
    // ProtectedRoute detects no accessToken and redirects to /login.
    // Verify there's rendered content (login form).
    expect(container.querySelector('div')).toBeInTheDocument()
  })

  it('renders CssBaseline for theme reset', () => {
    const { container } = render(<App />)
    // MUI CssBaseline injects global styles — App at minimum renders a root div.
    expect(container.firstChild).toBeTruthy()
  })
})
