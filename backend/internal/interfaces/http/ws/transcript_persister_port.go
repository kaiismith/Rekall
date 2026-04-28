package ws

import (
	"context"

	"github.com/rekall/backend/internal/application/services"
)

// TranscriptPersister is the narrow port through which the WS hub asks the
// application layer to record an ASR `final` segment. Defined here (instead
// of importing services.*TranscriptPersister directly) so the WS package can
// be tested with a fake; the concrete implementation in services satisfies
// this interface structurally.
type TranscriptPersister interface {
	RecordFinal(ctx context.Context, in services.RecordFinalInput) error
	CloseSession(ctx context.Context, in services.CloseSessionInput) error
}
