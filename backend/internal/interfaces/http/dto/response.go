package dto

// Response is the standard API response envelope.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// ErrorResponse is the standard API error envelope.
type ErrorResponse struct {
	Success bool      `json:"success" example:"false"`
	Error   ErrorBody `json:"error"`
}

// ErrorBody contains the machine-readable code and human-readable message.
type ErrorBody struct {
	Code    string      `json:"code"              example:"CALL_NOT_FOUND"`
	Message string      `json:"message"           example:"call with the given ID was not found"`
	Details interface{} `json:"details,omitempty"`
}

// Meta holds pagination metadata.
type Meta struct {
	Page    int `json:"page"     example:"1"`
	PerPage int `json:"per_page" example:"20"`
	Total   int `json:"total"    example:"150"`
}

// OK builds a successful response with optional data.
func OK(data interface{}) Response {
	return Response{Success: true, Data: data}
}

// Paginated builds a successful response with pagination metadata.
func Paginated(data interface{}, page, perPage, total int) Response {
	return Response{
		Success: true,
		Data:    data,
		Meta:    &Meta{Page: page, PerPage: perPage, Total: total},
	}
}

// Err builds an error response from code, message, and optional details.
func Err(code, message string, details interface{}) ErrorResponse {
	return ErrorResponse{
		Success: false,
		Error:   ErrorBody{Code: code, Message: message, Details: details},
	}
}

// ── Typed response envelopes (used only in swag annotations) ─────────────────
// These types give the spec concrete schemas instead of bare "object".

// MessageResponse is returned by endpoints that reply with a plain message.
type MessageResponse struct {
	Success bool   `json:"success" example:"true"`
	Data    MsgObj `json:"data"`
}

// MsgObj is the data payload of a MessageResponse.
type MsgObj struct {
	Message string `json:"message" example:"operation completed successfully"`
}

// AccessTokenResponse is returned by POST /auth/refresh.
type AccessTokenResponse struct {
	Success bool        `json:"success" example:"true"`
	Data    TokenObj    `json:"data"`
}

// TokenObj wraps a single access_token string.
type TokenObj struct {
	AccessToken string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// LoginResponseEnvelope wraps LoginResponse in the standard envelope.
type LoginResponseEnvelope struct {
	Success bool          `json:"success" example:"true"`
	Data    LoginResponse `json:"data"`
}

// UserResponseEnvelope wraps a single UserResponse.
type UserResponseEnvelope struct {
	Success bool         `json:"success" example:"true"`
	Data    UserResponse `json:"data"`
}

// UserListResponse wraps a paginated UserResponse slice.
type UserListResponse struct {
	Success bool           `json:"success" example:"true"`
	Data    []UserResponse `json:"data"`
	Meta    Meta           `json:"meta"`
}

// CallResponseEnvelope wraps a single CallResponse.
type CallResponseEnvelope struct {
	Success bool         `json:"success" example:"true"`
	Data    CallResponse `json:"data"`
}

// CallListResponse wraps a paginated CallResponse slice.
type CallListResponse struct {
	Success bool           `json:"success" example:"true"`
	Data    []CallResponse `json:"data"`
	Meta    Meta           `json:"meta"`
}

// OrgResponseEnvelope wraps a single OrgResponse.
type OrgResponseEnvelope struct {
	Success bool        `json:"success" example:"true"`
	Data    OrgResponse `json:"data"`
}

// OrgListResponse wraps an OrgResponse slice.
type OrgListResponse struct {
	Success bool          `json:"success" example:"true"`
	Data    []OrgResponse `json:"data"`
}

// MemberListResponse wraps a MemberResponse slice.
type MemberListResponse struct {
	Success bool             `json:"success" example:"true"`
	Data    []MemberResponse `json:"data"`
}

// DeptResponseEnvelope wraps a single DeptResponse.
type DeptResponseEnvelope struct {
	Success bool         `json:"success" example:"true"`
	Data    DeptResponse `json:"data"`
}

// DeptListResponse wraps a DeptResponse slice.
type DeptListResponse struct {
	Success bool           `json:"success" example:"true"`
	Data    []DeptResponse `json:"data"`
}

// DeptMemberListResponse wraps a DeptMemberResponse slice.
type DeptMemberListResponse struct {
	Success bool                 `json:"success" example:"true"`
	Data    []DeptMemberResponse `json:"data"`
}
