import { describe, it, expect } from 'vitest'
import {
  canAddDeptMember,
  canCreateOrg,
  canDeleteOrg,
  canManageDept,
  canManageOrg,
  canPromoteDeptMember,
  isPlatformAdmin,
} from '@/utils/permissions'
import type { User } from '@/types/auth'
import type { OrgMember, DeptMember } from '@/types/organization'

const admin: User = {
  id: '00000000-0000-0000-0000-000000000001',
  email: 'admin@rekall.io',
  full_name: 'Admin',
  role: 'admin',
  email_verified: true,
  created_at: '2026-04-25T00:00:00Z',
}
const member: User = { ...admin, id: '00000000-0000-0000-0000-000000000002', role: 'member' }

const owner: OrgMember = {
  user_id: 'u',
  org_id: 'o',
  role: 'owner',
  joined_at: '2026-04-25T00:00:00Z',
}
const orgAdmin: OrgMember = { ...owner, role: 'admin' }
const orgMember: OrgMember = { ...owner, role: 'member' }

const deptHead: DeptMember = {
  user_id: 'u',
  department_id: 'd',
  role: 'head',
  joined_at: '2026-04-25T00:00:00Z',
}
const deptMember: DeptMember = { ...deptHead, role: 'member' }

describe('isPlatformAdmin', () => {
  it.each([
    [admin, true],
    [member, false],
    [null, false],
    [undefined, false],
  ])('%o → %s', (u, expected) => {
    expect(isPlatformAdmin(u)).toBe(expected)
  })
})

describe('canCreateOrg', () => {
  it('only platform admins can create', () => {
    expect(canCreateOrg(admin)).toBe(true)
    expect(canCreateOrg(member)).toBe(false)
    expect(canCreateOrg(null)).toBe(false)
  })
})

describe('canManageOrg', () => {
  it('platform admin always wins', () => {
    expect(canManageOrg(null, admin)).toBe(true)
    expect(canManageOrg(orgMember, admin)).toBe(true)
  })
  it('owner and admin allowed', () => {
    expect(canManageOrg(owner, member)).toBe(true)
    expect(canManageOrg(orgAdmin, member)).toBe(true)
  })
  it('plain member denied', () => {
    expect(canManageOrg(orgMember, member)).toBe(false)
    expect(canManageOrg(null, member)).toBe(false)
  })
})

describe('canManageDept', () => {
  it('aliases canManageOrg', () => {
    expect(canManageDept(orgAdmin, member)).toBe(true)
    expect(canManageDept(orgMember, member)).toBe(false)
  })
})

describe('canAddDeptMember', () => {
  it('platform admin allowed regardless', () => {
    expect(canAddDeptMember(null, null, admin)).toBe(true)
  })
  it('org owner/admin allowed', () => {
    expect(canAddDeptMember(orgAdmin, deptMember, member)).toBe(true)
  })
  it('plain org member but dept head allowed', () => {
    expect(canAddDeptMember(orgMember, deptHead, member)).toBe(true)
  })
  it('plain everywhere denied', () => {
    expect(canAddDeptMember(orgMember, deptMember, member)).toBe(false)
    expect(canAddDeptMember(null, null, member)).toBe(false)
  })
})

describe('canPromoteDeptMember', () => {
  it('only org-level admins (or platform admins) — never dept heads', () => {
    expect(canPromoteDeptMember(orgAdmin, member)).toBe(true)
    expect(canPromoteDeptMember(null, admin)).toBe(true)
    expect(canPromoteDeptMember(orgMember, member)).toBe(false)
  })
})

describe('canDeleteOrg', () => {
  it('only the owner (or platform admin) can delete the org', () => {
    expect(canDeleteOrg(owner, member)).toBe(true)
    expect(canDeleteOrg(null, admin)).toBe(true)
  })
  it('org admin and plain member denied', () => {
    expect(canDeleteOrg(orgAdmin, member)).toBe(false)
    expect(canDeleteOrg(orgMember, member)).toBe(false)
    expect(canDeleteOrg(null, member)).toBe(false)
  })
})
