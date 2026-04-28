import { apiClient } from './api'
import type { ApiResponse, PaginatedResponse } from '@/types/common'
import type { Call, CreateCallPayload, ListCallsParams, UpdateCallPayload } from '@/types/call'
import type { Scope } from '@/types/scope'
import { buildQueryString } from '@/utils'
import { scopeToQueryParams } from '@/utils/scope'

export const callService = {
  /**
   * Fetch a paginated list of calls. Pass `scope` to restrict to an
   * organization, department, or open slice; defaults to the caller's full
   * visibility.
   */
  async list(params: ListCallsParams = {}, scope?: Scope | null): Promise<PaginatedResponse<Call>> {
    const merged: Record<string, unknown> = { ...params, ...scopeToQueryParams(scope ?? null) }
    const qs = buildQueryString(merged)
    const response = await apiClient.get<PaginatedResponse<Call>>(`/calls${qs}`)
    return response.data
  },

  /** Fetch a single call by ID. */
  async getById(id: string): Promise<ApiResponse<Call>> {
    const response = await apiClient.get<ApiResponse<Call>>(`/calls/${id}`)
    return response.data
  },

  /** Create a new call record. */
  async create(payload: CreateCallPayload): Promise<ApiResponse<Call>> {
    const response = await apiClient.post<ApiResponse<Call>>('/calls', payload)
    return response.data
  },

  /** Partially update a call. */
  async update(id: string, payload: UpdateCallPayload): Promise<ApiResponse<Call>> {
    const response = await apiClient.patch<ApiResponse<Call>>(`/calls/${id}`, payload)
    return response.data
  },

  /** Soft-delete a call. */
  async delete(id: string): Promise<void> {
    await apiClient.delete(`/calls/${id}`)
  },
}
