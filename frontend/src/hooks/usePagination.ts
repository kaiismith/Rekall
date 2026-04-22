import { useState } from 'react'
import { DEFAULT_PAGE, DEFAULT_PER_PAGE } from '@/constants'

interface UsePaginationOptions {
  initialPage?: number
  initialPerPage?: number
}

interface UsePaginationReturn {
  page: number
  perPage: number
  setPage: (page: number) => void
  setPerPage: (perPage: number) => void
  reset: () => void
  totalPages: (total: number) => number
}

/**
 * Manages pagination state for list views.
 */
export function usePagination(options: UsePaginationOptions = {}): UsePaginationReturn {
  const { initialPage = DEFAULT_PAGE, initialPerPage = DEFAULT_PER_PAGE } = options

  const [page, setPage] = useState(initialPage)
  const [perPage, setPerPage] = useState(initialPerPage)

  const handleSetPage = (newPage: number) => {
    if (newPage >= 1) setPage(newPage)
  }

  const handleSetPerPage = (newPerPage: number) => {
    setPerPage(newPerPage)
    setPage(1) // Reset to first page when changing page size
  }

  const reset = () => {
    setPage(initialPage)
    setPerPage(initialPerPage)
  }

  const totalPages = (total: number) => Math.max(1, Math.ceil(total / perPage))

  return { page, perPage, setPage: handleSetPage, setPerPage: handleSetPerPage, reset, totalPages }
}
