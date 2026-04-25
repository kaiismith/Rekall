package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	apperr "github.com/rekall/backend/pkg/errors"
	applogger "github.com/rekall/backend/pkg/logger"
	"github.com/rekall/backend/pkg/logger/catalog"
	"go.uber.org/zap"
)

// CallService orchestrates business logic for call management.
type CallService struct {
	callRepo       ports.CallRepository
	orgMemberRepo  ports.OrgMembershipRepository
	deptMemberRepo ports.DepartmentMembershipRepository
	logger         *zap.Logger
}

// NewCallService creates a CallService with its required dependencies.
// The logger is tagged with component="call_service" so every log line emitted
// here is automatically identified without repeating the tag at each call site.
//
// orgMemberRepo and deptMemberRepo may be nil in test setups that do not
// exercise scoped calls — scope validation short-circuits when the relevant
// repo is absent and scope is not requested.
func NewCallService(
	callRepo ports.CallRepository,
	orgMemberRepo ports.OrgMembershipRepository,
	deptMemberRepo ports.DepartmentMembershipRepository,
	logger *zap.Logger,
) *CallService {
	return &CallService{
		callRepo:       callRepo,
		orgMemberRepo:  orgMemberRepo,
		deptMemberRepo: deptMemberRepo,
		logger:         applogger.WithComponent(logger, "call_service"),
	}
}

// CreateCallInput holds the data required to create a new call record.
type CreateCallInput struct {
	UserID    uuid.UUID
	Title     string
	Metadata  map[string]interface{}
	ScopeType string // "organization" | "department" | ""
	ScopeID   *uuid.UUID
}

// UpdateCallInput holds the fields that may be updated on an existing call.
type UpdateCallInput struct {
	Title        *string
	Status       *string
	RecordingURL *string
	Transcript   *string
	StartedAt    *time.Time
	EndedAt      *time.Time
	Metadata     map[string]interface{}
}

// CreateCall validates and persists a new call record.
func (s *CallService) CreateCall(ctx context.Context, input CreateCallInput) (*entities.Call, error) {
	if input.Title == "" {
		catalog.CallValidationFailed.Warn(s.logger,
			zap.String("reason", "title is required"),
			zap.String("user_id", input.UserID.String()),
		)
		return nil, apperr.BadRequest("title is required")
	}
	if input.UserID == uuid.Nil {
		catalog.CallValidationFailed.Warn(s.logger,
			zap.String("reason", "user_id is required"),
		)
		return nil, apperr.BadRequest("user_id is required")
	}
	if input.Metadata == nil {
		input.Metadata = map[string]interface{}{}
	}

	// Scope coherence: both fields set or both empty; scope membership enforced.
	scopeSet := input.ScopeType != "" || input.ScopeID != nil
	if scopeSet {
		if input.ScopeType == "" || input.ScopeID == nil {
			return nil, apperr.BadRequest("scope_type and scope_id must be provided together")
		}
		if err := s.assertCallScopeMember(ctx, input.ScopeType, *input.ScopeID, input.UserID); err != nil {
			return nil, err
		}
	}

	now := time.Now().UTC()
	call := &entities.Call{
		ID:        uuid.New(),
		UserID:    input.UserID,
		Title:     input.Title,
		Status:    "pending",
		Metadata:  input.Metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if scopeSet {
		scopeType := input.ScopeType
		call.ScopeType = &scopeType
		call.ScopeID = input.ScopeID
	}

	created, err := s.callRepo.Create(ctx, call)
	if err != nil {
		catalog.CallCreateFailed.Error(s.logger,
			zap.Error(err),
			zap.String("user_id", input.UserID.String()),
			zap.String("title", input.Title),
		)
		return nil, apperr.Internal("failed to create call")
	}

	catalog.CallCreated.Info(s.logger,
		zap.String("call_id", created.ID.String()),
		zap.String("user_id", created.UserID.String()),
		zap.String("title", created.Title),
		zap.String("status", created.Status),
	)
	return created, nil
}

// GetCall retrieves a call by its ID.
func (s *CallService) GetCall(ctx context.Context, id uuid.UUID) (*entities.Call, error) {
	call, err := s.callRepo.GetByID(ctx, id)
	if err != nil {
		if apperr.IsNotFound(err) {
			catalog.CallNotFound.Warn(s.logger,
				zap.String("call_id", id.String()),
			)
			return nil, apperr.NotFound("Call", id.String())
		}
		catalog.CallGetFailed.Error(s.logger,
			zap.Error(err),
			zap.String("call_id", id.String()),
		)
		return nil, apperr.Internal("failed to retrieve call")
	}

	catalog.CallFetched.Debug(s.logger,
		zap.String("call_id", call.ID.String()),
		zap.String("status", call.Status),
	)
	return call, nil
}

// ListCalls returns a paginated list of calls, optionally filtered.
//
// When filter.Scope is non-nil and requests an org/dept slice, the caller must
// be a member — enforcement is layered on here so handlers do not need to
// duplicate the check.
func (s *CallService) ListCalls(
	ctx context.Context,
	callerID uuid.UUID,
	filter ports.ListCallsFilter,
	page, perPage int,
) ([]*entities.Call, int, error) {
	if filter.Scope != nil {
		switch filter.Scope.Kind {
		case ports.ScopeKindOrganization:
			if err := s.assertCallScopeMember(ctx, "organization", filter.Scope.ID, callerID); err != nil {
				return nil, 0, err
			}
		case ports.ScopeKindDepartment:
			if err := s.assertCallScopeMember(ctx, "department", filter.Scope.ID, callerID); err != nil {
				return nil, 0, err
			}
		case ports.ScopeKindOpen:
			// Open-scope visibility is caller-owned: constrain by user_id so a
			// user never sees another user's open items through a scope=open
			// filter.
			cid := callerID
			filter.UserID = &cid
		}
	}

	calls, total, err := s.callRepo.List(ctx, filter, page, perPage)
	if err != nil {
		catalog.CallListFailed.Error(s.logger,
			zap.Error(err),
			zap.Int("page", page),
			zap.Int("per_page", perPage),
		)
		return nil, 0, apperr.Internal("failed to list calls")
	}

	catalog.CallsListed.Debug(s.logger,
		zap.Int("count", len(calls)),
		zap.Int("total", total),
		zap.Int("page", page),
		zap.Int("per_page", perPage),
	)
	return calls, total, nil
}

// assertCallScopeMember confirms that userID is a member of the scope
// (organization or department) identified by scopeType/scopeID. Returns
// apperr.Forbidden when the user is not a member, or an Internal error if
// the lookup itself fails.
func (s *CallService) assertCallScopeMember(
	ctx context.Context,
	scopeType string,
	scopeID uuid.UUID,
	userID uuid.UUID,
) error {
	switch scopeType {
	case "organization":
		if s.orgMemberRepo == nil {
			return apperr.Internal("organization membership lookup unavailable")
		}
		mem, err := s.orgMemberRepo.GetByOrgAndUser(ctx, scopeID, userID)
		if err != nil {
			if apperr.IsNotFound(err) {
				return apperr.Forbidden("caller is not a member of the organization")
			}
			return apperr.Internal("failed to verify organization membership")
		}
		if mem == nil {
			return apperr.Forbidden("caller is not a member of the organization")
		}
		return nil
	case "department":
		if s.deptMemberRepo == nil {
			return apperr.Internal("department membership lookup unavailable")
		}
		mem, err := s.deptMemberRepo.GetByDeptAndUser(ctx, scopeID, userID)
		if err != nil {
			if apperr.IsNotFound(err) {
				return apperr.Forbidden("caller is not a member of the department")
			}
			return apperr.Internal("failed to verify department membership")
		}
		if mem == nil {
			return apperr.Forbidden("caller is not a member of the department")
		}
		return nil
	default:
		return apperr.BadRequest("scope_type must be 'organization' or 'department'")
	}
}

// UpdateCall applies a partial update to an existing call.
func (s *CallService) UpdateCall(ctx context.Context, id uuid.UUID, input UpdateCallInput) (*entities.Call, error) {
	call, err := s.GetCall(ctx, id)
	if err != nil {
		return nil, err
	}

	prevStatus := call.Status
	updatedFields := make([]string, 0, 6)

	if input.Title != nil {
		call.Title = *input.Title
		updatedFields = append(updatedFields, "title")
	}
	if input.Status != nil {
		call.Status = *input.Status
		updatedFields = append(updatedFields, "status")
	}
	if input.RecordingURL != nil {
		call.RecordingURL = input.RecordingURL
		updatedFields = append(updatedFields, "recording_url")
	}
	if input.Transcript != nil {
		call.Transcript = input.Transcript
		updatedFields = append(updatedFields, "transcript")
	}
	if input.StartedAt != nil {
		call.StartedAt = input.StartedAt
		updatedFields = append(updatedFields, "started_at")
	}
	if input.EndedAt != nil {
		call.EndedAt = input.EndedAt
		updatedFields = append(updatedFields, "ended_at")
		if call.StartedAt != nil {
			call.DurationSec = int(input.EndedAt.Sub(*call.StartedAt).Seconds())
			updatedFields = append(updatedFields, "duration_sec")
		}
	}
	if input.Metadata != nil {
		call.Metadata = input.Metadata
		updatedFields = append(updatedFields, "metadata")
	}
	call.UpdatedAt = time.Now().UTC()

	updated, err := s.callRepo.Update(ctx, call)
	if err != nil {
		catalog.CallUpdateFailed.Error(s.logger,
			zap.Error(err),
			zap.String("call_id", id.String()),
			zap.Strings("fields", updatedFields),
		)
		return nil, apperr.Internal("failed to update call")
	}

	catalog.CallUpdated.Info(s.logger,
		zap.String("call_id", updated.ID.String()),
		zap.String("user_id", updated.UserID.String()),
		zap.String("prev_status", prevStatus),
		zap.String("new_status", updated.Status),
		zap.Strings("updated_fields", updatedFields),
	)
	return updated, nil
}

// DeleteCall soft-deletes a call by ID.
func (s *CallService) DeleteCall(ctx context.Context, id uuid.UUID) error {
	call, err := s.GetCall(ctx, id)
	if err != nil {
		return err
	}

	if err := s.callRepo.SoftDelete(ctx, id); err != nil {
		catalog.CallDeleteFailed.Error(s.logger,
			zap.Error(err),
			zap.String("call_id", id.String()),
			zap.String("user_id", call.UserID.String()),
		)
		return apperr.Internal("failed to delete call")
	}

	catalog.CallDeleted.Info(s.logger,
		zap.String("call_id", id.String()),
		zap.String("user_id", call.UserID.String()),
		zap.String("title", call.Title),
		zap.String("status_at_deletion", call.Status),
	)
	return nil
}
