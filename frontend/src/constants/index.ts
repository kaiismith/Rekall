/** Base URL for the backend API. Injected by Vite at build time. */
export const API_BASE_URL = (import.meta.env['VITE_API_BASE_URL'] as string | undefined) ?? '/api/v1'

/** Default pagination values. */
export const DEFAULT_PAGE = 1
export const DEFAULT_PER_PAGE = 20

/** Route paths. */
export const ROUTES = {
  ROOT: '/',
  DASHBOARD: '/dashboard',
  CALLS: '/calls',
  CALL_DETAIL: '/calls/:id',
  // Auth
  LOGIN: '/login',
  REGISTER: '/register',
  FORGOT_PASSWORD: '/forgot-password',
  RESET_PASSWORD: '/reset-password',
  VERIFY_EMAIL: '/verify-email',
  // Organizations
  ORGANIZATIONS: '/organizations',
  ORG_DETAIL: '/organizations/:id',
  INVITATION_ACCEPT: '/invitations/accept',
  // Meetings
  MEETINGS: '/meetings',
  NEW_MEETING: '/meetings/new',
  MEETING_ROOM: '/meeting/:code',
  // Account
  PROFILE: '/profile',
  SETTINGS: '/settings',
  HELP: '/help',
  NOT_FOUND: '*',
} as const

/** Navigation items for the sidebar. */
export const NAV_ITEMS = [
  { label: 'Dashboard', path: ROUTES.DASHBOARD, icon: 'Dashboard' },
  { label: 'Calls', path: ROUTES.CALLS, icon: 'PhoneInTalk' },
  { label: 'Meetings', path: ROUTES.MEETINGS, icon: 'VideoCall' },
] as const

/** Call status display config. */
export const CALL_STATUS_CONFIG = {
  pending: { label: 'Pending', color: 'warning' as const },
  processing: { label: 'Processing', color: 'info' as const },
  done: { label: 'Done', color: 'success' as const },
  failed: { label: 'Failed', color: 'error' as const },
} as const

/** Sidebar width in pixels. */
export const SIDEBAR_WIDTH = 260
export const SIDEBAR_COLLAPSED_WIDTH = 72

/** Minimum meeting-code length accepted by the Join Meeting input. */
export const MIN_MEETING_CODE_LENGTH = 6
