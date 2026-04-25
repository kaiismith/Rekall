import { Fragment, useEffect } from 'react'
import { Box, Link, Skeleton, Typography } from '@mui/material'
import { Link as RouterLink, useLocation, useParams } from 'react-router-dom'
import { ROUTES, buildScopedRoute } from '@/constants'
import { useOrgsStore } from '@/store/orgsStore'
import { useDeptsStore } from '@/store/deptsStore'

interface Segment {
  /** Visible label, or null while still loading. */
  label: string | null
  /** Resolved target; absent for the trailing "current page" segment. */
  to?: string
}

/**
 * Build the breadcrumb segments from the route params + current pathname.
 * Pure (other than the synchronous store reads passed in by caller); a
 * separate function keeps it unit-testable in isolation.
 */
export function buildSegments(args: {
  pathname: string
  params: { id?: string; orgId?: string; deptId?: string }
  getOrgName: (id: string) => string | undefined
  getDeptName: (orgId: string, deptId: string) => string | undefined
}): Segment[] {
  const { pathname, params, getOrgName, getDeptName } = args
  const orgId = params.orgId ?? params.id
  const deptId = params.deptId

  const segs: Segment[] = [{ label: 'Organizations', to: ROUTES.ORGANIZATIONS }]

  if (!orgId) return segs
  segs.push({
    label: getOrgName(orgId) ?? null,
    to: deptId || isMeetingsOrCallsTail(pathname)
      ? buildScopedRoute.org(orgId)
      : undefined, // last segment when on /organizations/:id alone
  })

  if (deptId) {
    // Org → "Departments" (no destination beyond org tab) → Dept
    segs.push({
      label: 'Departments',
      to: `${buildScopedRoute.org(orgId)}?tab=departments`,
    })
    const tail = isMeetingsOrCallsTail(pathname)
    segs.push({
      label: getDeptName(orgId, deptId) ?? null,
      to: tail ? buildScopedRoute.dept(orgId, deptId) : undefined,
    })
    if (tail === 'meetings') segs.push({ label: 'Meetings' })
    else if (tail === 'calls') segs.push({ label: 'Calls' })
    return segs
  }

  const tail = isMeetingsOrCallsTail(pathname)
  if (tail === 'meetings') segs.push({ label: 'Meetings' })
  else if (tail === 'calls') segs.push({ label: 'Calls' })
  return segs
}

function isMeetingsOrCallsTail(pathname: string): 'meetings' | 'calls' | null {
  if (pathname.endsWith('/meetings')) return 'meetings'
  if (pathname.endsWith('/calls')) return 'calls'
  return null
}

/**
 * Renders the scope breadcrumb beneath the TopBar on any scoped page.
 * Reads org/dept params from the URL and resolves names via the stores.
 */
export function ScopeBreadcrumb() {
  const params = useParams<{ id?: string; orgId?: string; deptId?: string }>()
  const { pathname } = useLocation()
  const getOrgName = useOrgsStore((s) => s.getOrgName)
  const orgs = useOrgsStore((s) => s.orgs)
  const loadOrgs = useOrgsStore((s) => s.load)
  const ensureDeptsLoaded = useDeptsStore((s) => s.ensureLoaded)
  const getDeptName = useDeptsStore((s) => s.getDeptName)

  useEffect(() => {
    if (orgs === null) void loadOrgs()
  }, [orgs, loadOrgs])

  const orgId = params.orgId ?? params.id
  useEffect(() => {
    if (params.deptId && orgId) void ensureDeptsLoaded(orgId)
  }, [params.deptId, orgId, ensureDeptsLoaded])

  const segments = buildSegments({ pathname, params, getOrgName, getDeptName })

  return (
    <Box
      component="nav"
      aria-label="Breadcrumb"
      sx={{
        display: 'flex',
        alignItems: 'center',
        flexWrap: 'wrap',
        gap: 0.5,
        px: { xs: 2, sm: 3 },
        py: 1.25,
        fontSize: '0.875rem',
      }}
    >
      {segments.map((seg, i) => {
        const isLast = i === segments.length - 1
        return (
          <Fragment key={i}>
            {i > 0 && (
              <Typography
                component="span"
                aria-hidden
                sx={{ color: 'text.disabled', mx: 0.5, fontSize: '0.95rem', lineHeight: 1 }}
              >
                ›
              </Typography>
            )}
            {seg.label === null ? (
              <Skeleton variant="text" width={64} height={20} />
            ) : seg.to && !isLast ? (
              <Link
                component={RouterLink}
                to={seg.to}
                underline="hover"
                color="text.secondary"
                sx={{ fontWeight: 500 }}
              >
                {seg.label}
              </Link>
            ) : (
              <Typography
                component="span"
                color={isLast ? 'text.primary' : 'text.secondary'}
                aria-current={isLast ? 'page' : undefined}
                sx={{ fontWeight: isLast ? 600 : 500 }}
              >
                {seg.label}
              </Typography>
            )}
          </Fragment>
        )
      })}
    </Box>
  )
}
