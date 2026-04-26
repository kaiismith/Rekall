package dto

import "time"

// ASRSessionRequest is the optional body for POST /calls/:id/asr-session.
// All fields default sensibly; callers may omit the body entirely.
type ASRSessionRequest struct {
	ModelID    string `json:"model_id,omitempty"    binding:"omitempty,max=64"`
	Language   string `json:"language,omitempty"    binding:"omitempty,max=16"`
	TTLSeconds uint32 `json:"ttl_seconds,omitempty" binding:"omitempty,gte=60,lte=300"`
}

// ASRSessionPayload is the data envelope returned on success.
type ASRSessionPayload struct {
	SessionID    string    `json:"session_id"`
	SessionToken string    `json:"session_token"`
	WsURL        string    `json:"ws_url"`
	ExpiresAt    time.Time `json:"expires_at"`
	ModelID      string    `json:"model_id"`
	SampleRate   int32     `json:"sample_rate"`
	FrameFormat  string    `json:"frame_format"`
}

// ASRSessionResponseEnvelope is the envelope used by /calls/:id/asr-session.
type ASRSessionResponseEnvelope struct {
	Success bool              `json:"success"`
	Data    ASRSessionPayload `json:"data"`
}

// ASRSessionEndRequest carries the session_id to terminate. The call_id is
// in the URL path so it stays consistent with the issue endpoint.
type ASRSessionEndRequest struct {
	SessionID string `json:"session_id" binding:"required,uuid"`
}

// ASRSessionEndPayload returns the stitched final transcript so the frontend
// can persist it to the call record.
type ASRSessionEndPayload struct {
	FinalTranscript string `json:"final_transcript"`
	FinalCount      uint32 `json:"final_count"`
}

// ASRSessionEndResponseEnvelope wraps the end-session response.
type ASRSessionEndResponseEnvelope struct {
	Success bool                 `json:"success"`
	Data    ASRSessionEndPayload `json:"data"`
}
