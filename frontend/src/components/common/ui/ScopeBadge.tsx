import { useEffect } from 'react'
import { Chip, Skeleton } from '@mui/material'
import { matchPath, useLocation, useNavigate } from 'react-router-dom'
import type { Scope } from '@/types/scope'
import { useOrgsStore } from '@/store/orgsStore'
import { useDeptsStore } from '@/store/deptsStore'
import { buildScopedRoute } from '@/constants'

interface ScopeBadgeProps {
  scope: Scope
  /** Override the auto-detected "already on a scoped page" state. */
  forceNonClickable?: boolean
}

/**
 * Compact label that says where a meeting or call lives:
 *   `Open` — neutral, never clickable
 *   `{OrgName}` — clickable; jumps to /organizations/:id/meetings
 *   `{OrgName} › {DeptName}` — clickable; jumps to the dept-scoped meetings list
 *
 * On a scoped page the badge becomes non-clickable — the user is already there.
 */
export function ScopeBadge({ scope, forceNonClickable }: ScopeBadgeProps) {
  const navigate = useNavigate()
  const orgs = useOrgsStore((s) => s.orgs)
  const loadOrgs = useOrgsStore((s) => s.load)
  const getOrgName = useOrgsStore((s) => s.getOrgName)
  const ensureDeptsLoaded = useDeptsStore((s) => s.ensureLoaded)
  const getDeptName = useDeptsStore((s) => s.getDeptName)
  const { pathname } = useLocation()

  // Lazy-trigger orgs load so the badge resolves on first render of any
  // page that has not already booted the store.
  useEffect(() => {
    if (orgs === null) void loadOrgs()
  }, [orgs, loadOrgs])

  // For dept badges, ensure that org's depts are in flight or cached.
  useEffect(() => {
    if (scope.type === 'department') void ensureDeptsLoaded(scope.orgId)
  }, [scope, ensureDeptsLoaded])

  const isOnScopedPage = matchPath('/organizations/:id/*', pathname) !== null

  let label: string | null = null
  if (scope.type === 'open') {
    label = 'Open'
  } else if (scope.type === 'organization') {
    const name = getOrgName(scope.id)
    if (name) label = name
  } else {
    const orgName = getOrgName(scope.orgId)
    const deptName = getDeptName(scope.orgId, scope.id)
    if (orgName && deptName) label = `${orgName} › ${deptName}`
  }

  if (label === null) return <Skeleton variant="rounded" width={80} height={22} />

  const clickable = !forceNonClickable && !isOnScopedPage && scope.type !== 'open'

  const target = (() => {
    if (scope.type === 'organization') return buildScopedRoute.orgMeetings(scope.id)
    if (scope.type === 'department') return buildScopedRoute.deptMeetings(scope.orgId, scope.id)
    return null
  })()

  return (
    <Chip
      label={label}
      size="small"
      aria-label={label}
      onClick={
        clickable && target
          ? (e) => {
              e.stopPropagation()
              navigate(target)
            }
          : undefined
      }
      sx={{
        bgcolor: scope.type === 'open' ? 'rgba(255,255,255,0.06)' : 'rgba(129,140,248,0.14)',
        color: scope.type === 'open' ? 'text.secondary' : 'primary.light',
        fontSize: '0.72rem',
        fontWeight: 600,
        letterSpacing: '0.01em',
        height: 22,
        cursor: clickable ? 'pointer' : 'default',
        maxWidth: 240,
        '& .MuiChip-label': { px: 1, overflow: 'hidden', textOverflow: 'ellipsis' },
      }}
    />
  )
}
