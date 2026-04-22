import { apiClient } from './api'
import type { ApiResponse } from '@/types/common'
import type {
  User,
  LoginResponse,
  RegisterPayload,
  LoginPayload,
  ForgotPasswordPayload,
  ResetPasswordPayload,
  ResendVerificationPayload,
} from '@/types/auth'

const BASE = '/auth'

export const authService = {
  register: async (payload: RegisterPayload): Promise<User> => {
    const { data } = await apiClient.post<ApiResponse<User>>(`${BASE}/register`, payload)
    return data.data
  },

  login: async (payload: LoginPayload): Promise<LoginResponse> => {
    const { data } = await apiClient.post<ApiResponse<LoginResponse>>(`${BASE}/login`, payload)
    return data.data
  },

  logout: async (): Promise<void> => {
    await apiClient.post(`${BASE}/logout`)
  },

  refresh: async (): Promise<{ access_token: string }> => {
    const { data } = await apiClient.post<ApiResponse<{ access_token: string }>>(`${BASE}/refresh`)
    return data.data
  },

  me: async (): Promise<User> => {
    const { data } = await apiClient.get<ApiResponse<User>>(`${BASE}/me`)
    return data.data
  },

  verifyEmail: async (token: string): Promise<void> => {
    await apiClient.get(`${BASE}/verify`, { params: { token } })
  },

  resendVerification: async (payload: ResendVerificationPayload): Promise<void> => {
    await apiClient.post(`${BASE}/verify/resend`, payload)
  },

  forgotPassword: async (payload: ForgotPasswordPayload): Promise<void> => {
    await apiClient.post(`${BASE}/password/forgot`, payload)
  },

  resetPassword: async (payload: ResetPasswordPayload): Promise<void> => {
    await apiClient.post(`${BASE}/password/reset`, payload)
  },
}
