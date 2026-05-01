package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rekall/backend/internal/domain/ports"
	"github.com/rekall/backend/internal/interfaces/http/dto"
	"github.com/rekall/backend/internal/interfaces/http/handlers"
)

// stubGenerator is a minimal ports.NoteGenerator used to drive the handler
// test through every (configured, auth_mode) shape.
type stubGenerator struct {
	configured bool
	authMode   string
	modelID    string
	provider   string
}

func (s *stubGenerator) Generate(_ context.Context, _ ports.NoteGeneratorInput, _ ports.StreamCallback) (*ports.NoteGeneratorOutput, error) {
	return nil, nil
}
func (s *stubGenerator) ModelID() string    { return s.modelID }
func (s *stubGenerator) AuthMode() string   { return s.authMode }
func (s *stubGenerator) Provider() string   { return s.provider }
func (s *stubGenerator) IsConfigured() bool { return s.configured }

func katServeHealth(t *testing.T, h *handlers.KatHandler) (int, dto.KatHealthResponse, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/healthz/kat", h.Health)
	req := httptest.NewRequest(http.MethodGet, "/healthz/kat", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	body, _ := readKatBody(rec)
	return rec.Code, body, rec.Body.String()
}

func readKatBody(rec *httptest.ResponseRecorder) (dto.KatHealthResponse, error) {
	var resp dto.KatHealthResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	return resp, err
}

func TestKatHandler_Health_APIKeyMode(t *testing.T) {
	gen := &stubGenerator{configured: true, authMode: "api_key", modelID: "gpt-4o-mini"}
	h := handlers.NewKatHandler(gen, "https://my-foundry.openai.azure.com/some/path")

	code, body, raw := katServeHealth(t, h)
	require.Equal(t, http.StatusOK, code)
	assert.True(t, body.Configured)
	assert.Equal(t, "api_key", body.AuthMode)
	assert.Equal(t, "gpt-4o-mini", body.Deployment)
	assert.Equal(t, "my-foundry.openai.azure.com", body.EndpointHost,
		"only host should be surfaced; the path is stripped")
	// Defensive: the actual API key value (configured at boot) MUST never
	// appear anywhere in the response body. The handler does not see it; we
	// only assert that no fields like "api_key": "<value>" sneak in.
	assert.False(t, strings.Contains(raw, "secret-"),
		"response body must not leak any API key material")
	assert.False(t, strings.Contains(raw, "\"api_key\":\""),
		"response body must not include an api_key field")
}

func TestKatHandler_Health_ManagedIdentityMode(t *testing.T) {
	gen := &stubGenerator{configured: true, authMode: "managed_identity", modelID: "gpt-4o"}
	h := handlers.NewKatHandler(gen, "https://prod.openai.azure.com")

	code, body, _ := katServeHealth(t, h)
	require.Equal(t, http.StatusOK, code)
	assert.True(t, body.Configured)
	assert.Equal(t, "managed_identity", body.AuthMode)
	assert.Equal(t, "gpt-4o", body.Deployment)
	assert.Equal(t, "prod.openai.azure.com", body.EndpointHost)
}

func TestKatHandler_Health_Unconfigured(t *testing.T) {
	gen := &stubGenerator{configured: false, authMode: "none", modelID: ""}
	h := handlers.NewKatHandler(gen, "")

	code, body, _ := katServeHealth(t, h)
	require.Equal(t, http.StatusOK, code)
	assert.False(t, body.Configured)
	assert.Equal(t, "none", body.AuthMode)
	assert.Equal(t, "", body.Deployment)
	assert.Equal(t, "", body.EndpointHost)
}

func TestKatHandler_Health_NilGenerator(t *testing.T) {
	// When KAT_ENABLED=false the wiring may pass nil for the generator.
	// The handler must still return 200 with "configured=false".
	h := handlers.NewKatHandler(nil, "")
	code, body, _ := katServeHealth(t, h)
	require.Equal(t, http.StatusOK, code)
	assert.False(t, body.Configured)
	assert.Equal(t, "none", body.AuthMode)
}
