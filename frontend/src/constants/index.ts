/** Base URL for the backend API. Injected by Vite at build time. */
export const API_BASE_URL =
  (import.meta.env['VITE_API_BASE_URL'] as string | undefined) ?? '/api/v1'

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
  // Organizations + scoped surfaces
  ORGANIZATIONS: '/organizations',
  ORG_DETAIL: '/organizations/:id',
  ORG_MEETINGS: '/organizations/:id/meetings',
  ORG_CALLS: '/organizations/:id/calls',
  ORG_DEPT_DETAIL: '/organizations/:orgId/departments/:deptId',
  ORG_DEPT_MEETINGS: '/organizations/:orgId/departments/:deptId/meetings',
  ORG_DEPT_CALLS: '/organizations/:orgId/departments/:deptId/calls',
  INVITATION_ACCEPT: '/invitations/accept',
  // Records (formerly "Meetings" tab) — list + detail. The live-room route
  // /meeting/:code is the WebRTC surface and remains separate.
  RECORDS: '/records',
  NEW_RECORD: '/records/new',
  RECORD_DETAIL: '/records/:code',
  // Meetings — deprecated aliases redirecting to RECORDS during the deploy
  // window. Remove in a follow-up once external links have migrated.
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
  { label: 'Records', path: ROUTES.RECORDS, icon: 'PhoneInTalk' },
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

/**
 * Helpers that resolve scoped route templates with concrete IDs. Components
 * SHALL go through these instead of building paths inline so the URL shape
 * lives in one place.
 */
export const buildScopedRoute = {
  org: (id: string) => `/organizations/${id}`,
  orgMeetings: (id: string) => `/organizations/${id}/meetings`,
  orgCalls: (id: string) => `/organizations/${id}/calls`,
  dept: (orgId: string, deptId: string) => `/organizations/${orgId}/departments/${deptId}`,
  deptMeetings: (orgId: string, deptId: string) =>
    `/organizations/${orgId}/departments/${deptId}/meetings`,
  deptCalls: (orgId: string, deptId: string) =>
    `/organizations/${orgId}/departments/${deptId}/calls`,
}

/** RFC-4122 UUID — used to validate dynamic route params before issuing API calls. */
const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
export function isUuid(s: string | undefined | null): s is string {
  return typeof s === 'string' && UUID_RE.test(s)
}
