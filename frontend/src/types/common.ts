/** Standard paginated API response. */
export interface PaginatedResponse<T> {
  success: boolean
  data: T[]
  meta: PaginationMeta
}

/** Standard single-item API response. */
export interface ApiResponse<T> {
  success: boolean
  data: T
}

/** Pagination metadata from the API. */
export interface PaginationMeta {
  page: number
  per_page: number
  total: number
}

/** Standard API error response. */
export interface ApiErrorResponse {
  success: false
  error: {
    code: string
    message: string
    details?: unknown
  }
}

/** Query params for paginated list endpoints. */
export interface PaginationParams {
  page?: number
  per_page?: number
}
