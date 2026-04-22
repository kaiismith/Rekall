import { describe, it, expect } from 'vitest'
import { ApiError } from '@/services/api'

// ── ApiError ──────────────────────────────────────────────────────────────────
// The response interceptor logic lives in authStore (which imports apiClient),
// making it hard to unit-test in isolation without circular mock complexity.
// These tests cover the ApiError class directly, which is the shared error type
// used throughout all service and interceptor code.

describe('ApiError', () => {
  it('is an instance of Error', () => {
    const err = new ApiError('NOT_FOUND', 'Not found', 404)
    expect(err).toBeInstanceOf(Error)
  })

  it('sets name to "ApiError"', () => {
    const err = new ApiError('NOT_FOUND', 'Not found', 404)
    expect(err.name).toBe('ApiError')
  })

  it('stores code, message, and status', () => {
    const err = new ApiError('VALIDATION_ERROR', 'Invalid input', 422)
    expect(err.code).toBe('VALIDATION_ERROR')
    expect(err.message).toBe('Invalid input')
    expect(err.status).toBe(422)
  })

  it('stores optional details', () => {
    const details = { field: 'email', reason: 'invalid format' }
    const err = new ApiError('VALIDATION_ERROR', 'Bad', 422, details)
    expect(err.details).toEqual(details)
  })

  it('details defaults to undefined when not provided', () => {
    const err = new ApiError('SERVER_ERROR', 'Oops', 500)
    expect(err.details).toBeUndefined()
  })

  it('can be caught as an Error', () => {
    expect(() => {
      throw new ApiError('TIMEOUT', 'Timed out', 408)
    }).toThrow('Timed out')
  })

  it('status 0 represents a network-level error', () => {
    const err = new ApiError('NETWORK_ERROR', 'Network error. Check your connection.', 0)
    expect(err.status).toBe(0)
    expect(err.code).toBe('NETWORK_ERROR')
  })

  it('status 408 represents a timeout', () => {
    const err = new ApiError('TIMEOUT', 'Request timed out. Please try again.', 408)
    expect(err.status).toBe(408)
  })

  it('status 401 represents an unauthenticated error', () => {
    const err = new ApiError('UNAUTHENTICATED', 'Session expired.', 401)
    expect(err.status).toBe(401)
    expect(err.code).toBe('UNAUTHENTICATED')
  })
})
