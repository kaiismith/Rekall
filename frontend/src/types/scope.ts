/**
 * Scope identifies where a meeting or call lives — either an organization,
 * a department within an organization, or "open" (no team attachment).
 *
 * The discriminated union enforces that the right ID is present for the right
 * variant: department scopes carry both their own id AND the parent orgId so
 * components can render breadcrumbs without an extra lookup.
 */
export type Scope =
  | { type: 'open' }
  | { type: 'organization'; id: string }
  | { type: 'department'; id: string; orgId: string }

/** Singleton open-scope value. Use this rather than allocating per-render. */
export const OPEN_SCOPE: Scope = { type: 'open' }

/** Type guard for the open variant. */
export function isOpenScope(s: Scope | null | undefined): s is { type: 'open' } {
  return s?.type === 'open'
}

/** Type guard for the organization variant. */
export function isOrgScope(s: Scope | null | undefined): s is { type: 'organization'; id: string } {
  return s?.type === 'organization'
}

/** Type guard for the department variant. */
export function isDeptScope(
  s: Scope | null | undefined,
): s is { type: 'department'; id: string; orgId: string } {
  return s?.type === 'department'
}
