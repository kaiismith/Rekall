import type { Scope } from '@/types/scope'

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

/**
 * Parse a Scope value from the `?scope=` URL query parameter.
 *
 * Accepted forms:
 *   `scope=open`              → `{ type: 'open' }`
 *   `scope=org:<uuid>`        → `{ type: 'organization', id }`
 *   `scope=dept:<orgUuid>:<deptUuid>` → `{ type: 'department', orgId, id }`
 *
 * Returns `null` when the parameter is missing or malformed — callers should
 * treat that as "no scope filter" and fall back to defaults.
 */
export function parseScopeFromUrl(params: URLSearchParams): Scope | null {
  const raw = params.get('scope')
  if (!raw) return null

  if (raw === 'open') return { type: 'open' }

  if (raw.startsWith('org:')) {
    const id = raw.slice(4)
    if (!UUID_RE.test(id)) return null
    return { type: 'organization', id }
  }

  if (raw.startsWith('dept:')) {
    const rest = raw.slice(5)
    const colon = rest.indexOf(':')
    if (colon < 0) return null
    const orgId = rest.slice(0, colon)
    const id = rest.slice(colon + 1)
    if (!UUID_RE.test(orgId) || !UUID_RE.test(id)) return null
    return { type: 'department', orgId, id }
  }

  return null
}

/**
 * Serialise a Scope as the value of the `?scope=` URL parameter.
 * The inverse of `parseScopeFromUrl`.
 */
export function scopeToUrlParam(scope: Scope): string {
  switch (scope.type) {
    case 'open':
      return 'open'
    case 'organization':
      return `org:${scope.id}`
    case 'department':
      return `dept:${scope.orgId}:${scope.id}`
  }
}

/**
 * Translate a Scope into the wire-level `filter[scope_type]` /
 * `filter[scope_id]` query-parameter pair the backend list endpoints expect.
 */
export function scopeToQueryParams(scope: Scope | null): Record<string, string> {
  if (!scope) return {}
  if (scope.type === 'open') return { 'filter[scope_type]': 'open' }
  if (scope.type === 'organization') {
    return { 'filter[scope_type]': 'organization', 'filter[scope_id]': scope.id }
  }
  return { 'filter[scope_type]': 'department', 'filter[scope_id]': scope.id }
}

/** True if `a` and `b` describe the same scope. */
export function scopesEqual(a: Scope | null, b: Scope | null): boolean {
  if (a === b) return true
  if (!a || !b) return false
  if (a.type !== b.type) return false
  if (a.type === 'open') return true
  if (a.type === 'organization' && b.type === 'organization') return a.id === b.id
  if (a.type === 'department' && b.type === 'department') {
    return a.id === b.id && a.orgId === b.orgId
  }
  return false
}
