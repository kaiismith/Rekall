import { describe, it, expect, vi } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useApi } from '@/hooks/useApi'
import { ApiError } from '@/services/api'

describe('useApi', () => {
  // ── Initial state ─────────────────────────────────────────────────────────────

  it('starts with data null, loading false, error null', () => {
    const { result } = renderHook(() => useApi(vi.fn()))
    expect(result.current.data).toBeNull()
    expect(result.current.loading).toBe(false)
    expect(result.current.error).toBeNull()
  })

  // ── Loading state ─────────────────────────────────────────────────────────────

  it('sets loading true while the request is in flight', async () => {
    let resolve!: (v: string) => void
    const fn = vi.fn(() => new Promise<string>((r) => { resolve = r }))

    const { result } = renderHook(() => useApi(fn))

    act(() => { result.current.execute() })

    expect(result.current.loading).toBe(true)
    expect(result.current.data).toBeNull()
    expect(result.current.error).toBeNull()

    await act(async () => { resolve('done') })
  })

  // ── Success ───────────────────────────────────────────────────────────────────

  it('sets data and clears loading after a successful call', async () => {
    const fn = vi.fn().mockResolvedValue({ id: 1, name: 'Alice' })

    const { result } = renderHook(() => useApi(fn))

    await act(async () => { await result.current.execute() })

    expect(result.current.data).toEqual({ id: 1, name: 'Alice' })
    expect(result.current.loading).toBe(false)
    expect(result.current.error).toBeNull()
  })

  it('returns the resolved value from execute()', async () => {
    const fn = vi.fn().mockResolvedValue(42)

    const { result } = renderHook(() => useApi(fn))

    let returnValue: number | null = null
    await act(async () => { returnValue = await result.current.execute() })

    expect(returnValue).toBe(42)
  })

  it('passes arguments through to the wrapped function', async () => {
    const fn = vi.fn().mockResolvedValue('ok')

    const { result } = renderHook(() => useApi(fn))

    await act(async () => { await result.current.execute('arg1', 99) })

    expect(fn).toHaveBeenCalledWith('arg1', 99)
  })

  // ── Error — ApiError ──────────────────────────────────────────────────────────

  it('stores the ApiError and clears loading when the call rejects with ApiError', async () => {
    const apiError = new ApiError('NOT_FOUND', 'resource not found', 404)
    const fn = vi.fn().mockRejectedValue(apiError)

    const { result } = renderHook(() => useApi(fn))

    await act(async () => { await result.current.execute() })

    expect(result.current.error).toBe(apiError)
    expect(result.current.loading).toBe(false)
    expect(result.current.data).toBeNull()
  })

  it('returns null from execute() when the call rejects', async () => {
    const fn = vi.fn().mockRejectedValue(new ApiError('BAD_REQUEST', 'bad', 400))

    const { result } = renderHook(() => useApi(fn))

    let returnValue: unknown = 'sentinel'
    await act(async () => { returnValue = await result.current.execute() })

    expect(returnValue).toBeNull()
  })

  // ── Error — generic (non-ApiError) ────────────────────────────────────────────

  it('wraps non-ApiError rejections in a generic ApiError', async () => {
    const fn = vi.fn().mockRejectedValue(new Error('Network timeout'))

    const { result } = renderHook(() => useApi(fn))

    await act(async () => { await result.current.execute() })

    expect(result.current.error).toBeInstanceOf(ApiError)
    expect(result.current.error?.message).toMatch(/unexpected error/i)
  })

  // ── reset ─────────────────────────────────────────────────────────────────────

  it('clears data, error, and loading when reset() is called', async () => {
    const fn = vi.fn().mockResolvedValue('value')

    const { result } = renderHook(() => useApi(fn))

    await act(async () => { await result.current.execute() })
    expect(result.current.data).toBe('value')

    act(() => { result.current.reset() })

    expect(result.current.data).toBeNull()
    expect(result.current.error).toBeNull()
    expect(result.current.loading).toBe(false)
  })

  it('clears error state when reset() is called after a failed call', async () => {
    const fn = vi.fn().mockRejectedValue(new ApiError('ERR', 'oops', 500))

    const { result } = renderHook(() => useApi(fn))

    await act(async () => { await result.current.execute() })
    expect(result.current.error).not.toBeNull()

    act(() => { result.current.reset() })

    expect(result.current.error).toBeNull()
  })
})
