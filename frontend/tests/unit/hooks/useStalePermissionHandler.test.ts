import { describe, it, expect, vi } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useStalePermissionHandler } from '@/hooks/useStalePermissionHandler'
import { ApiError } from '@/services/api'

describe('useStalePermissionHandler', () => {
  it('handles 403 ApiError: notify + invalidate fire, returns true', () => {
    const invalidate = vi.fn()
    const notify = vi.fn()
    const { result } = renderHook(() => useStalePermissionHandler({ invalidate, notify }))

    const handled = result.current(new ApiError('FORBIDDEN', 'nope', 403))

    expect(handled).toBe(true)
    expect(invalidate).toHaveBeenCalledOnce()
    expect(notify).toHaveBeenCalledOnce()
    expect(notify.mock.calls[0]![0]).toMatch(/no longer have permission/i)
  })

  it('handles 403 axios-shaped error', () => {
    const invalidate = vi.fn()
    const { result } = renderHook(() => useStalePermissionHandler({ invalidate }))

    const handled = result.current({ response: { status: 403 } })

    expect(handled).toBe(true)
    expect(invalidate).toHaveBeenCalledOnce()
  })

  it('passes through non-403 errors untouched, returns false', () => {
    const invalidate = vi.fn()
    const notify = vi.fn()
    const { result } = renderHook(() => useStalePermissionHandler({ invalidate, notify }))

    const handled = result.current(new ApiError('SERVER_ERROR', 'boom', 500))

    expect(handled).toBe(false)
    expect(invalidate).not.toHaveBeenCalled()
    expect(notify).not.toHaveBeenCalled()
  })
})
