import { describe, it, expect } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { usePagination } from '@/hooks/usePagination'

describe('usePagination', () => {
  // ── Initial state ───────────────────────────────────────────────────────────

  it('starts on page 1 with the default per-page value', () => {
    const { result } = renderHook(() => usePagination())
    expect(result.current.page).toBe(1)
    expect(result.current.perPage).toBeGreaterThan(0)
  })

  it('accepts custom initial page and perPage', () => {
    const { result } = renderHook(() => usePagination({ initialPage: 3, initialPerPage: 50 }))
    expect(result.current.page).toBe(3)
    expect(result.current.perPage).toBe(50)
  })

  // ── setPage ─────────────────────────────────────────────────────────────────

  it('updates page when setPage is called with a valid value', () => {
    const { result } = renderHook(() => usePagination())
    act(() => result.current.setPage(5))
    expect(result.current.page).toBe(5)
  })

  it('ignores setPage calls with 0', () => {
    const { result } = renderHook(() => usePagination())
    act(() => result.current.setPage(0))
    expect(result.current.page).toBe(1)
  })

  it('ignores setPage calls with negative numbers', () => {
    const { result } = renderHook(() => usePagination())
    act(() => result.current.setPage(-3))
    expect(result.current.page).toBe(1)
  })

  it('allows setPage(1) — boundary value is accepted', () => {
    const { result } = renderHook(() => usePagination())
    act(() => result.current.setPage(3))
    act(() => result.current.setPage(1))
    expect(result.current.page).toBe(1)
  })

  // ── setPerPage ──────────────────────────────────────────────────────────────

  it('updates perPage when setPerPage is called', () => {
    const { result } = renderHook(() => usePagination())
    act(() => result.current.setPerPage(50))
    expect(result.current.perPage).toBe(50)
  })

  it('resets page to 1 when perPage changes', () => {
    const { result } = renderHook(() => usePagination())
    act(() => result.current.setPage(4))
    act(() => result.current.setPerPage(50))
    expect(result.current.page).toBe(1)
  })

  // ── reset ───────────────────────────────────────────────────────────────────

  it('resets page and perPage to initial values', () => {
    const { result } = renderHook(() => usePagination({ initialPage: 2, initialPerPage: 10 }))
    act(() => result.current.setPage(7))
    act(() => result.current.setPerPage(50))
    act(() => result.current.reset())
    expect(result.current.page).toBe(2)
    expect(result.current.perPage).toBe(10)
  })

  it('reset with defaults returns to page 1 and default perPage', () => {
    const { result } = renderHook(() => usePagination())
    act(() => result.current.setPage(9))
    act(() => result.current.reset())
    expect(result.current.page).toBe(1)
  })

  // ── totalPages ──────────────────────────────────────────────────────────────

  it('calculates totalPages correctly for even division', () => {
    const { result } = renderHook(() => usePagination({ initialPerPage: 10 }))
    expect(result.current.totalPages(100)).toBe(10)
  })

  it('rounds up totalPages when items do not divide evenly', () => {
    const { result } = renderHook(() => usePagination({ initialPerPage: 10 }))
    expect(result.current.totalPages(101)).toBe(11)
  })

  it('returns at least 1 page even for 0 total items', () => {
    const { result } = renderHook(() => usePagination({ initialPerPage: 20 }))
    expect(result.current.totalPages(0)).toBe(1)
  })

  it('returns 1 page when total equals perPage exactly', () => {
    const { result } = renderHook(() => usePagination({ initialPerPage: 20 }))
    expect(result.current.totalPages(20)).toBe(1)
  })

  it('returns 1 page when total is less than perPage', () => {
    const { result } = renderHook(() => usePagination({ initialPerPage: 20 }))
    expect(result.current.totalPages(5)).toBe(1)
  })
})
