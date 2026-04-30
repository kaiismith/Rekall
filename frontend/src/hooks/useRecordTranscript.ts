import { useMemo } from 'react'
import { useInfiniteQuery } from '@tanstack/react-query'

import { transcriptService } from '@/services/transcriptService'
import type {
  MeetingTranscriptResponse,
  TranscriptSegmentDTO,
  TranscriptSessionDTO,
} from '@/types/transcript'

const DEFAULT_PER_PAGE = 50

export interface UseRecordTranscriptResult {
  segments: TranscriptSegmentDTO[]
  sessions: TranscriptSessionDTO[]
  isLoading: boolean
  isError: boolean
  error: unknown
  isFetchingNextPage: boolean
  hasNextPage: boolean
  loadMore: () => void
  total: number
}

/**
 * Paginated reader for a record's stored transcript. Wraps useInfiniteQuery
 * over GET /meetings/:code/transcript and exposes a flat `segments` slice
 * accumulated across every loaded page. Sessions are taken from the most
 * recent page (the backend returns the same session list on every page).
 *
 * Auto-load on scroll is intentionally NOT implemented in v1 — the timeline
 * component renders an explicit "Load more" affordance and calls `loadMore`.
 */
export function useRecordTranscript(
  meetingCode: string,
  perPage: number = DEFAULT_PER_PAGE,
): UseRecordTranscriptResult {
  const query = useInfiniteQuery<MeetingTranscriptResponse>({
    queryKey: ['record-transcript', meetingCode, perPage],
    enabled: Boolean(meetingCode),
    initialPageParam: 1,
    queryFn: ({ pageParam }) =>
      transcriptService.getMeetingTranscript(meetingCode, {
        page: pageParam as number,
        per_page: perPage,
      }),
    getNextPageParam: (lastPage) =>
      lastPage.pagination.has_more ? lastPage.pagination.page + 1 : undefined,
  })

  const segments = useMemo(() => query.data?.pages.flatMap((p) => p.segments) ?? [], [query.data])

  const lastPage = query.data?.pages[query.data.pages.length - 1]
  const sessions = lastPage?.sessions ?? []
  const total = lastPage?.pagination.total ?? 0

  return {
    segments,
    sessions,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    isFetchingNextPage: query.isFetchingNextPage,
    hasNextPage: Boolean(query.hasNextPage),
    loadMore: () => {
      if (query.hasNextPage && !query.isFetchingNextPage) {
        void query.fetchNextPage()
      }
    },
    total,
  }
}
