package foundry

import (
	"net/url"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/azure"
	"github.com/openai/openai-go/v3/option"
	"go.uber.org/zap"

	"github.com/rekall/backend/pkg/logger/catalog"
)

// Auth modes recorded by the client at boot. The KAT_FOUNDRY_INITIALIZED log
// line carries one of these as the auth_mode field.
const (
	AuthModeAPIKey          = "api_key"
	AuthModeManagedIdentity = "managed_identity"
	AuthModeNone            = "none"
)

// Provider identifies which LLM backend the Kat note generator talks to.
// Both providers ride on the same openai-go SDK underneath; they differ only
// in the `option.RequestOption` set used at client construction.
type Provider string

const (
	// ProviderFoundry routes through Azure AI Foundry (api-key OR managed
	// identity via DefaultAzureCredential).
	ProviderFoundry Provider = "foundry"
	// ProviderOpenAI routes through OpenAI's public API (api-key only). An
	// optional `BaseURL` may point at any OpenAI-wire-compatible endpoint
	// (vLLM, LM Studio, an Azure OpenAI-compatible proxy, etc.).
	ProviderOpenAI Provider = "openai"
)

// Config carries the env-bound knobs for the Kat AI provider client.
// Construction is boot-time and immutable; a config change requires a
// process restart.
type Config struct {
	// Provider selects the backend: "foundry" or "openai". Empty defaults to
	// "foundry" for backward compatibility with the v1 single-provider shape.
	Provider Provider

	// Foundry fields (used when Provider == ProviderFoundry).
	Endpoint   string // e.g. https://my-foundry.openai.azure.com
	Deployment string // e.g. gpt-4o-mini
	APIVersion string // e.g. 2024-08-01-preview
	APIKey     string // empty => use DefaultAzureCredential

	// OpenAI fields (used when Provider == ProviderOpenAI).
	OpenAIAPIKey  string // required for the openai provider
	OpenAIBaseURL string // empty = api.openai.com; set for compatible providers
	OpenAIModel   string // e.g. gpt-4o-mini

	RequestTimeout time.Duration // per-call deadline; shared across providers
}

// Client wraps the underlying openai-go Client with the provider + auth
// bookkeeping the rest of the Kat stack needs. Construction never fails the
// boot — a misconfigured client is returned in a degraded state with
// Configured() == false.
type Client struct {
	oai         *openai.Client // nil when configured == false
	provider    Provider       // selected backend; "" when not configured
	deployment  string         // model id reported via /healthz/kat
	apiVersion  string
	authMode    string
	configured  bool
	notReadyErr error
	timeout     time.Duration
}

// NewClient selects a provider + auth strategy based on cfg and returns a
// usable Client or a degraded one (Configured() == false). The boot log line
// is emitted as a side effect; the API key value is never logged.
func NewClient(cfg Config, log *zap.Logger) *Client {
	provider := cfg.Provider
	if provider == "" {
		provider = ProviderFoundry
	}

	switch provider {
	case ProviderOpenAI:
		return newOpenAIClient(cfg, log)
	case ProviderFoundry:
		return newFoundryClient(cfg, log)
	default:
		catalog.KatFoundryUnconfigured.Warn(log,
			zap.String("reason", "unknown_provider"),
			zap.String("provider", string(provider)),
		)
		return &Client{authMode: AuthModeNone, configured: false, notReadyErr: ErrFoundryUnconfigured}
	}
}

// newFoundryClient constructs the Azure AI Foundry path: api-key OR
// DefaultAzureCredential. Identical to v1 behaviour.
func newFoundryClient(cfg Config, log *zap.Logger) *Client {
	if cfg.Endpoint == "" {
		catalog.KatFoundryUnconfigured.Warn(log,
			zap.String("provider", string(ProviderFoundry)),
			zap.String("reason", "missing_endpoint"))
		return &Client{authMode: AuthModeNone, configured: false, notReadyErr: ErrFoundryUnconfigured}
	}
	if cfg.Deployment == "" {
		catalog.KatFoundryUnconfigured.Warn(log,
			zap.String("provider", string(ProviderFoundry)),
			zap.String("reason", "missing_deployment"))
		return &Client{authMode: AuthModeNone, configured: false, notReadyErr: ErrFoundryUnconfigured}
	}

	apiVersion := cfg.APIVersion
	if apiVersion == "" {
		apiVersion = "2024-08-01-preview"
	}

	if cfg.APIKey != "" {
		c := openai.NewClient(
			azure.WithEndpoint(cfg.Endpoint, apiVersion),
			azure.WithAPIKey(cfg.APIKey),
			option.WithMaxRetries(0),
		)
		catalog.KatFoundryInitialized.Info(log,
			zap.String("provider", string(ProviderFoundry)),
			zap.String("auth_mode", AuthModeAPIKey),
			zap.String("endpoint_host", hostOnly(cfg.Endpoint)),
			zap.String("deployment", cfg.Deployment),
			zap.String("api_version", apiVersion),
		)
		return &Client{
			oai:        &c,
			provider:   ProviderFoundry,
			deployment: cfg.Deployment,
			apiVersion: apiVersion,
			authMode:   AuthModeAPIKey,
			configured: true,
			timeout:    cfg.RequestTimeout,
		}
	}

	// No API key => DefaultAzureCredential chain.
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		catalog.KatFoundryUnconfigured.Warn(log,
			zap.String("provider", string(ProviderFoundry)),
			zap.String("reason", "credential_construct_failed"),
			zap.Error(err),
		)
		return &Client{authMode: AuthModeNone, configured: false, notReadyErr: err}
	}

	return newClientWithCredential(cfg, apiVersion, cred, log)
}

// newOpenAIClient constructs the OpenAI path. API-key auth only (managed
// identity is Azure-specific). The optional BaseURL lets callers point at
// any OpenAI-wire-compatible endpoint.
func newOpenAIClient(cfg Config, log *zap.Logger) *Client {
	if cfg.OpenAIAPIKey == "" {
		catalog.KatFoundryUnconfigured.Warn(log,
			zap.String("provider", string(ProviderOpenAI)),
			zap.String("reason", "missing_api_key"))
		return &Client{authMode: AuthModeNone, configured: false, notReadyErr: ErrFoundryUnconfigured}
	}
	if cfg.OpenAIModel == "" {
		catalog.KatFoundryUnconfigured.Warn(log,
			zap.String("provider", string(ProviderOpenAI)),
			zap.String("reason", "missing_model"))
		return &Client{authMode: AuthModeNone, configured: false, notReadyErr: ErrFoundryUnconfigured}
	}

	opts := []option.RequestOption{
		option.WithAPIKey(cfg.OpenAIAPIKey),
		option.WithMaxRetries(0),
	}
	endpointHost := "api.openai.com"
	if cfg.OpenAIBaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.OpenAIBaseURL))
		endpointHost = hostOnly(cfg.OpenAIBaseURL)
	}

	c := openai.NewClient(opts...)
	catalog.KatFoundryInitialized.Info(log,
		zap.String("provider", string(ProviderOpenAI)),
		zap.String("auth_mode", AuthModeAPIKey),
		zap.String("endpoint_host", endpointHost),
		zap.String("model", cfg.OpenAIModel),
	)
	return &Client{
		oai:        &c,
		provider:   ProviderOpenAI,
		deployment: cfg.OpenAIModel,
		authMode:   AuthModeAPIKey,
		configured: true,
		timeout:    cfg.RequestTimeout,
	}
}

// newClientWithCredential is split out so unit tests can inject a fake
// azcore.TokenCredential without exercising the full DefaultAzureCredential
// chain.
func newClientWithCredential(cfg Config, apiVersion string, cred azcore.TokenCredential, log *zap.Logger) *Client {
	c := openai.NewClient(
		azure.WithEndpoint(cfg.Endpoint, apiVersion),
		azure.WithTokenCredential(cred),
		option.WithMaxRetries(0),
	)
	catalog.KatFoundryInitialized.Info(log,
		zap.String("provider", string(ProviderFoundry)),
		zap.String("auth_mode", AuthModeManagedIdentity),
		zap.String("endpoint_host", hostOnly(cfg.Endpoint)),
		zap.String("deployment", cfg.Deployment),
		zap.String("api_version", apiVersion),
	)
	return &Client{
		oai:        &c,
		provider:   ProviderFoundry,
		deployment: cfg.Deployment,
		apiVersion: apiVersion,
		authMode:   AuthModeManagedIdentity,
		configured: true,
		timeout:    cfg.RequestTimeout,
	}
}

// Configured reports whether Generate may succeed at all. False when
// construction couldn't pick any auth strategy.
func (c *Client) Configured() bool { return c.configured }

// AuthMode returns the recorded auth mode for /healthz/kat and logging.
func (c *Client) AuthMode() string { return c.authMode }

// Provider returns the selected backend ("foundry" | "openai" | "" when not
// configured). Surfaced via /healthz/kat so the frontend footer can label
// the provider.
func (c *Client) Provider() string {
	if !c.configured {
		return ""
	}
	return string(c.provider)
}

// Deployment returns the canonical model deployment name (== ModelID for the
// NoteGenerator port).
func (c *Client) Deployment() string { return c.deployment }

// hostOnly returns the host component of a URL, falling back to the input
// string verbatim if it can't be parsed. Used to keep boot logs from leaking
// the resource path.
func hostOnly(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return rawURL
	}
	return u.Host
}
