import { describe, it, expect } from 'vitest'
import { buildSegments } from '@/components/common/ui/ScopeBreadcrumb'

const ORG = '00000000-0000-0000-0000-00000000a001'
const DEPT = '00000000-0000-0000-0000-00000000d001'

describe('buildSegments', () => {
  const getOrgName = (id: string) => (id === ORG ? 'Acme' : undefined)
  const getDeptName = (orgId: string, deptId: string) =>
    orgId === ORG && deptId === DEPT ? 'Engineering' : undefined

  it('returns just "Organizations" when no params are present', () => {
    const segs = buildSegments({ pathname: '/organizations', params: {}, getOrgName, getDeptName })
    expect(segs).toEqual([{ label: 'Organizations', to: '/organizations' }])
  })

  it('renders org name as the trailing segment on /organizations/:id', () => {
    const segs = buildSegments({
      pathname: `/organizations/${ORG}`,
      params: { id: ORG },
      getOrgName,
      getDeptName,
    })
    expect(segs).toHaveLength(2)
    expect(segs[1]).toMatchObject({ label: 'Acme' })
    // Last segment has no `to` (it's the current page).
    expect(segs[1]!.to).toBeUndefined()
  })

  it('appends "Meetings" trailing segment on org-meetings route', () => {
    const segs = buildSegments({
      pathname: `/organizations/${ORG}/meetings`,
      params: { id: ORG },
      getOrgName,
      getDeptName,
    })
    expect(segs.map((s) => s.label)).toEqual(['Organizations', 'Acme', 'Meetings'])
    // Org segment is now linkable since "Meetings" is the trailing one.
    expect(segs[1]!.to).toBe(`/organizations/${ORG}`)
  })

  it('renders "Calls" trailing segment on org-calls route', () => {
    const segs = buildSegments({
      pathname: `/organizations/${ORG}/calls`,
      params: { id: ORG },
      getOrgName,
      getDeptName,
    })
    expect(segs.map((s) => s.label)).toEqual(['Organizations', 'Acme', 'Calls'])
  })

  it('renders the dept route with org → Departments → dept name → Meetings/Calls tail', () => {
    const segs = buildSegments({
      pathname: `/organizations/${ORG}/departments/${DEPT}/meetings`,
      params: { orgId: ORG, deptId: DEPT },
      getOrgName,
      getDeptName,
    })
    expect(segs.map((s) => s.label)).toEqual([
      'Organizations',
      'Acme',
      'Departments',
      'Engineering',
      'Meetings',
    ])
    // The Departments segment links into the org's departments tab.
    expect(segs[2]!.to).toBe(`/organizations/${ORG}?tab=departments`)
  })

  it('renders null label when the org name is still loading', () => {
    const segs = buildSegments({
      pathname: `/organizations/${ORG}/meetings`,
      params: { id: ORG },
      getOrgName: () => undefined, // still loading
      getDeptName,
    })
    expect(segs[1]!.label).toBeNull()
  })

  it('renders null label when the dept name is still loading', () => {
    const segs = buildSegments({
      pathname: `/organizations/${ORG}/departments/${DEPT}`,
      params: { orgId: ORG, deptId: DEPT },
      getOrgName,
      getDeptName: () => undefined, // still loading
    })
    // Last segment is the dept, which is unresolved.
    expect(segs[segs.length - 1]!.label).toBeNull()
  })
})
