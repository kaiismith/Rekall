/** Public representation of a user (matches backend UserResponse DTO). */
export interface User {
  id: string
  email: string
  full_name: string
  role: string
  email_verified: boolean
  created_at: string
}

/** Response from POST /auth/login and POST /auth/refresh. */
export interface LoginResponse {
  access_token: string
  user: User
}

/** Body for POST /auth/register. */
export interface RegisterPayload {
  email: string
  password: string
  full_name: string
}

/** Body for POST /auth/login. */
export interface LoginPayload {
  email: string
  password: string
}

/** Body for POST /auth/password/forgot. */
export interface ForgotPasswordPayload {
  email: string
}

/** Body for POST /auth/password/reset. */
export interface ResetPasswordPayload {
  token: string
  password: string
}

/** Body for POST /auth/verify/resend. */
export interface ResendVerificationPayload {
  email: string
}
