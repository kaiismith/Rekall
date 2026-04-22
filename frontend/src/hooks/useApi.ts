import { useState, useCallback } from 'react'
import { ApiError } from '@/services/api'

interface UseApiState<T> {
  data: T | null
  loading: boolean
  error: ApiError | null
}

interface UseApiReturn<T, Args extends unknown[]> extends UseApiState<T> {
  execute: (...args: Args) => Promise<T | null>
  reset: () => void
}

/**
 * A thin wrapper for imperative API calls (mutations, one-off fetches).
 * For data-fetching with caching, prefer TanStack React Query hooks directly.
 */
export function useApi<T, Args extends unknown[]>(
  fn: (...args: Args) => Promise<T>,
): UseApiReturn<T, Args> {
  const [state, setState] = useState<UseApiState<T>>({
    data: null,
    loading: false,
    error: null,
  })

  const execute = useCallback(
    async (...args: Args): Promise<T | null> => {
      setState({ data: null, loading: true, error: null })
      try {
        const result = await fn(...args)
        setState({ data: result, loading: false, error: null })
        return result
      } catch (err) {
        const apiError =
          err instanceof ApiError
            ? err
            : new ApiError('UNKNOWN', 'An unexpected error occurred', 0)
        setState({ data: null, loading: false, error: apiError })
        return null
      }
    },
    [fn],
  )

  const reset = useCallback(() => {
    setState({ data: null, loading: false, error: null })
  }, [])

  return { ...state, execute, reset }
}
