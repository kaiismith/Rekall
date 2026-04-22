import { useSearchParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { meetingService } from '@/services/meetingService'
import type { Meeting, MeetingStatusFilter, MeetingSortKey } from '@/types/meeting'

const DEFAULT_SORT: MeetingSortKey = 'created_at_desc'

export function useMeetingsList() {
  const [searchParams, setSearchParams] = useSearchParams()

  const status = (searchParams.get('status') as MeetingStatusFilter | null) ?? undefined
  const sort = (searchParams.get('sort') as MeetingSortKey | null) ?? DEFAULT_SORT

  const { data, isLoading, isError } = useQuery({
    queryKey: ['meetings', 'list', status, sort],
    queryFn: () => meetingService.listMine({ status, sort }),
  })

  const meetings: Meeting[] = data?.data ?? []

  function setStatus(value: MeetingStatusFilter | undefined) {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      if (value) {
        next.set('status', value)
      } else {
        next.delete('status')
      }
      return next
    })
  }

  function setSort(value: MeetingSortKey) {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      if (value === DEFAULT_SORT) {
        next.delete('sort')
      } else {
        next.set('sort', value)
      }
      return next
    })
  }

  const activeFilterCount = status ? 1 : 0

  return { meetings, isLoading, isError, status, sort, setStatus, setSort, activeFilterCount }
}
