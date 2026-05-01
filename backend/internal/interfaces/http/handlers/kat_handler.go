package handlers

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"github.com/rekall/backend/internal/domain/ports"
	"github.com/rekall/backend/internal/interfaces/http/dto"
)

// KatHandler exposes the Kat liveness probe at /healthz/kat. It carries no
// list endpoints — Kat notes are not persisted, so there is nothing to read
// over HTTP. The frontend uses the WS late-join replay (driven by
// KatNotesService.OnParticipantJoined) for history.
type KatHandler struct {
	gen          ports.NoteGenerator
	endpointHost string
}

// NewKatHandler wires a NoteGenerator into the handler. endpoint is the URL
// surfaced via /healthz/kat (host only — no path, no resource id) so
// operators can sanity-check which deployment the backend is talking to.
// Pass "" to omit; the OpenAI provider falls back to "api.openai.com".
func NewKatHandler(gen ports.NoteGenerator, endpoint string) *KatHandler {
	host := endpoint
	if u, err := url.Parse(endpoint); err == nil && u.Host != "" {
		host = u.Host
	}
	return &KatHandler{gen: gen, endpointHost: host}
}

// Health serves GET /healthz/kat. Public (no auth) — used by liveness probes
// and by the frontend bootstrap to decide whether to render the live or
// offline panel. Returns the auth mode + deployment + endpoint host. Never
// includes the API key or any token.
//
// @Summary      Kat liveness probe
// @Description  Returns whether Kat (the live AI notes assistant) is configured. Public; never includes any secret.
// @Tags         Health
// @Produce      json
// @Success      200  {object}  dto.KatHealthResponse  "Probe response"
// @Router       /healthz/kat [get]
func (h *KatHandler) Health(c *gin.Context) {
	resp := dto.KatHealthResponse{
		Configured:   false,
		Provider:     "",
		AuthMode:     "none",
		Deployment:   "",
		EndpointHost: h.endpointHost,
	}
	if h.gen != nil {
		resp.Configured = h.gen.IsConfigured()
		resp.Provider = h.gen.Provider()
		resp.AuthMode = h.gen.AuthMode()
		resp.Deployment = h.gen.ModelID()
	}
	c.JSON(http.StatusOK, resp)
}
