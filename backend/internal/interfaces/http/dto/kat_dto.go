package dto

// KatHealthResponse is the body of GET /healthz/kat. Returned at the top level
// (not wrapped in the standard {data: ...} envelope) so liveness probes don't
// have to peek into a sub-object. The response MUST NOT include the API key
// or any token; only the provider, auth mode, and host are surfaced.
type KatHealthResponse struct {
	Configured   bool   `json:"configured"`
	Provider     string `json:"provider"`  // "foundry" | "openai" | "" (unconfigured)
	AuthMode     string `json:"auth_mode"` // "api_key" | "managed_identity" | "none"
	Deployment   string `json:"deployment"`
	EndpointHost string `json:"endpoint_host"`
}
