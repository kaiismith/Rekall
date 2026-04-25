import type { User } from '@/types/auth'
import type { OrgMember, DeptMember } from '@/types/organization'

/**
 * Client-side permission predicates.
 *
 * These are UI affordances only — the server enforces every gate via the
 * routes/middleware/services. Components consume these so we don't render
 * buttons that will simply 403 when clicked.
 *
 * Every predicate is null-safe: pass `null` for a missing user / membership
 * and the predicate returns `false`.
 */

/** True when the user holds the platform-level admin role. */
export const isPlatformAdmin = (u: User | null | undefined): boolean =>
  u?.role === 'admin'

/**
 * Only platform admins create organizations. Regular users are added to orgs
 * by an admin via invite or owner_email; they don't bootstrap their own.
 */
export const canCreateOrg = (u: User | null | undefined): boolean =>
  isPlatformAdmin(u)

/**
 * Manage the organization itself — rename, delete, create/edit/delete
 * departments, invite members, change member roles. Org owner OR admin
 * qualifies; platform admins always qualify (fallthrough).
 */
export function canManageOrg(
  membership: OrgMember | null | undefined,
  user: User | null | undefined,
): boolean {
  if (isPlatformAdmin(user)) return true
  return membership?.role === 'owner' || membership?.role === 'admin'
}

/**
 * Manage the department metadata (rename, delete) — same predicate as
 * canManageOrg today, named separately so future divergence stays surgical.
 */
export const canManageDept = canManageOrg

/**
 * Add or remove members on a specific department. Org owner/admin and
 * platform admins always qualify. A plain org member who is the dept's head
 * also qualifies.
 */
export function canAddDeptMember(
  orgMembership: OrgMember | null | undefined,
  deptMembership: DeptMember | null | undefined,
  user: User | null | undefined,
): boolean {
  if (canManageOrg(orgMembership, user)) return true
  return deptMembership?.role === 'head'
}

/**
 * Promote a department member to head, or demote a head back to member —
 * leadership is decided at the org level. Dept heads cannot promote each
 * other.
 */
export const canPromoteDeptMember = canManageOrg
