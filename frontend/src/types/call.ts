export type CallStatus = 'pending' | 'processing' | 'done' | 'failed'
export type CallScopeType = 'organization' | 'department'

export interface Call {
  id: string
  user_id: string
  title: string
  duration_sec: number
  status: CallStatus
  recording_url?: string
  transcript?: string
  metadata: Record<string, unknown>
  scope_type?: CallScopeType
  scope_id?: string
  started_at?: string
  ended_at?: string
  created_at: string
  updated_at: string
}

export interface CreateCallPayload {
  user_id: string
  title: string
  metadata?: Record<string, unknown>
}

export interface UpdateCallPayload {
  title?: string
  status?: CallStatus
  recording_url?: string
  transcript?: string
  started_at?: string
  ended_at?: string
  metadata?: Record<string, unknown>
}

export interface ListCallsParams {
  page?: number
  per_page?: number
  user_id?: string
  status?: CallStatus
}
