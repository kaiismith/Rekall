// Package foundry provides the Azure AI Foundry adapter for the Kat live-notes
// scheduler. It selects between API-key and DefaultAzureCredential auth at
// boot, exposes a NoteGenerator implementation, and never persists anything.
package foundry

import "errors"

// ErrFoundryUnconfigured is returned by Generate when the client could not
// construct any auth strategy at boot. Callers should NOT retry; the operator
// must set KAT_FOUNDRY_ENDPOINT / DEPLOYMENT and either an API key or a
// reachable managed-identity chain.
var ErrFoundryUnconfigured = errors.New("foundry: client not configured")

// ErrFoundryRateLimited is returned after a second 429 response. The Kat
// scheduler responds by entering error-cooldown; the request path itself does
// not retry further.
var ErrFoundryRateLimited = errors.New("foundry: rate limited")

// ErrFoundryUnavailable is returned after a second 5xx response, on a context
// deadline, or on any transport error.
var ErrFoundryUnavailable = errors.New("foundry: unavailable")

// ErrFoundryParseFailed is returned when the model's reply could not be
// parsed as the Kat JSON schema, even after one corrective retry.
var ErrFoundryParseFailed = errors.New("foundry: response parse failed")
