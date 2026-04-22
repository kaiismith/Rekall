import '@testing-library/jest-dom'
import { cleanup } from '@testing-library/react'
import { afterEach, vi } from 'vitest'

// Clean up after each test to prevent DOM leaks
afterEach(() => {
  cleanup()
})

// Silence MUI prop-type warnings in test output
vi.stubGlobal('console', {
  ...console,
  error: vi.fn(),
  warn: vi.fn(),
})
