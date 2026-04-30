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

// Config carries the env-bound knobs for the Foundry client. Construction is
// boot-time and immutable; a config change requires a process restart.
type Config struct {
	Endpoint       string        // e.g. https://my-foundry.openai.azure.com
	Deployment     string        // e.g. gpt-4o-mini
	APIVersion     string        // e.g. 2024-08-01-preview
	APIKey         string        // empty => use DefaultAzureCredential
	RequestTimeout time.Duration // per-call deadline; default applied by caller
}

// Client wraps the underlying openai-go Client (configured against an Azure
// Foundry endpoint) with the auth-strategy bookkeeping the rest of the Kat
// stack needs. Construction never fails the boot — a misconfigured client is
// returned in a degraded state with Configured() == false.
type Client struct {
	oai         *openai.Client // nil when configured == false
	deployment  string
	apiVersion  string
	authMode    string
	configured  bool
	notReadyErr error
	timeout     time.Duration
}

// NewClient selects an auth strategy based on cfg and returns a usable Client
// or a degraded one (Configured() == false). The boot log line is emitted as
// a side effect; the API key value is never logged.
func NewClient(cfg Config, log *zap.Logger) *Client {
	if cfg.Endpoint == "" {
		catalog.KatFoundryUnconfigured.Warn(log,
			zap.String("reason", "missing_endpoint"))
		return &Client{authMode: AuthModeNone, configured: false, notReadyErr: ErrFoundryUnconfigured}
	}
	if cfg.Deployment == "" {
		catalog.KatFoundryUnconfigured.Warn(log,
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
			zap.String("auth_mode", AuthModeAPIKey),
			zap.String("endpoint_host", hostOnly(cfg.Endpoint)),
			zap.String("deployment", cfg.Deployment),
			zap.String("api_version", apiVersion),
		)
		return &Client{
			oai:        &c,
			deployment: cfg.Deployment,
			apiVersion: apiVersion,
			authMode:   AuthModeAPIKey,
			configured: true,
			timeout:    cfg.RequestTimeout,
		}
	}

	// No API key => DefaultAzureCredential chain
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		catalog.KatFoundryUnconfigured.Warn(log,
			zap.String("reason", "credential_construct_failed"),
			zap.Error(err),
		)
		return &Client{authMode: AuthModeNone, configured: false, notReadyErr: err}
	}

	return newClientWithCredential(cfg, apiVersion, cred, log)
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
		zap.String("auth_mode", AuthModeManagedIdentity),
		zap.String("endpoint_host", hostOnly(cfg.Endpoint)),
		zap.String("deployment", cfg.Deployment),
		zap.String("api_version", apiVersion),
	)
	return &Client{
		oai:        &c,
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
