import { useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useInfiniteQuery } from '@tanstack/react-query'
import { meetingService } from '@/services/meetingService'
import type {
  Meeting,
  MeetingStatusFilter,
  MeetingSortKey,
  PaginatedMeetingListResponse,
} from '@/types/meeting'
import type { Scope } from '@/types/scope'
import { parseScopeFromUrl, scopeToUrlParam, scopesEqual } from '@/utils/scope'

const DEFAULT_SORT: MeetingSortKey = 'created_at_desc'

/** Page size used for the records/meetings list — five rows before "Show more". */
export const MEETINGS_PAGE_SIZE = 5

interface UseMeetingsListOptions {
  /** When set, the scope is fixed by the caller (used by Scoped Meetings Page); the URL is ignored. */
  forcedScope?: Scope | null
  /** Override the default page size (mostly for tests). */
  perPage?: number
}

/**
 * Paginated meetings list — wraps useInfiniteQuery over GET /meetings/mine.
 * The hook flattens every loaded page into a single `meetings` array, exposes
 * `hasMore` / `loadMore`, and surfaces the server's reported `total`.
 */
export function useMeetingsList(options: UseMeetingsListOptions = {}) {
  const [searchParams, setSearchParams] = useSearchParams()
  const perPage = options.perPage ?? MEETINGS_PAGE_SIZE

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

  const query = useInfiniteQuery<PaginatedMeetingListResponse>({
    queryKey: ['meetings', 'list', status, sort, scopeKey, perPage],
    initialPageParam: 1,
    queryFn: ({ pageParam }) =>
      meetingService.listMine(
        { status, sort, page: pageParam as number, per_page: perPage },
        scope,
      ),
    getNextPageParam: (lastPage) =>
      lastPage.pagination.has_more ? lastPage.pagination.page + 1 : undefined,
  })

  const meetings: Meeting[] = useMemo(
    () => query.data?.pages.flatMap((p) => p.data) ?? [],
    [query.data],
  )

  const lastPage = query.data?.pages[query.data.pages.length - 1]
  const total = lastPage?.pagination.total ?? 0

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

  const activeFilterCount =
    (status ? 1 : 0) + (scope !== null && options.forcedScope == null ? 1 : 0)

  return {
    meetings,
    isLoading: query.isLoading,
    isError: query.isError,
    isFetchingNextPage: query.isFetchingNextPage,
    hasNextPage: Boolean(query.hasNextPage),
    loadMore: () => {
      if (query.hasNextPage && !query.isFetchingNextPage) {
        void query.fetchNextPage()
      }
    },
    total,
    status,
    sort,
    scope,
    setStatus,
    setSort,
    setScope,
    activeFilterCount,
  }
}
