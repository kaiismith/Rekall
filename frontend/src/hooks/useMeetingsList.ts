import { useSearchParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { meetingService } from '@/services/meetingService'
import type { Meeting, MeetingStatusFilter, MeetingSortKey } from '@/types/meeting'
import type { Scope } from '@/types/scope'
import { parseScopeFromUrl, scopeToUrlParam, scopesEqual } from '@/utils/scope'

const DEFAULT_SORT: MeetingSortKey = 'created_at_desc'

interface UseMeetingsListOptions {
  /** When set, the scope is fixed by the caller (used by Scoped Meetings Page); the URL is ignored. */
  forcedScope?: Scope | null
}

export function useMeetingsList(options: UseMeetingsListOptions = {}) {
  const [searchParams, setSearchParams] = useSearchParams()

  const status = (searchParams.get('status') as MeetingStatusFilter | null) ?? undefined
  const sort = (searchParams.get('sort') as MeetingSortKey | null) ?? DEFAULT_SORT
  const urlScope = parseScopeFromUrl(searchParams)
  const scope = options.forcedScope ?? urlScope

  const scopeKey =
    scope === null
      ? 'none'
      : scope.type === 'open'
        ? 'open'
        : scope.type === 'organization'
          ? `org:${scope.id}`
          : `dept:${scope.orgId}:${scope.id}`

  const { data, isLoading, isError } = useQuery({
    queryKey: ['meetings', 'list', status, sort, scopeKey],
    queryFn: () => meetingService.listMine({ status, sort }, scope),
  })

  const meetings: Meeting[] = data?.data ?? []

  function setStatus(value: MeetingStatusFilter | undefined) {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      if (value) next.set('status', value)
      else next.delete('status')
      return next
    })
  }

  function setSort(value: MeetingSortKey) {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      if (value === DEFAULT_SORT) next.delete('sort')
      else next.set('sort', value)
      return next
    })
  }

  function setScope(value: Scope | null) {
    if (scopesEqual(value, urlScope)) return
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      if (value === null) next.delete('scope')
      else next.set('scope', scopeToUrlParam(value))
      return next
    })
  }

  const activeFilterCount = (status ? 1 : 0) + (scope !== null && options.forcedScope == null ? 1 : 0)

  return {
    meetings,
    isLoading,
    isError,
    status,
    sort,
    scope,
    setStatus,
    setSort,
    setScope,
    activeFilterCount,
  }
}
