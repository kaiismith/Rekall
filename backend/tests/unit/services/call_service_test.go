package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rekall/backend/internal/application/services"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	apperr "github.com/rekall/backend/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ─── Mock ─────────────────────────────────────────────────────────────────────

type mockCallRepo struct{ mock.Mock }

func (m *mockCallRepo) Create(ctx context.Context, call *entities.Call) (*entities.Call, error) {
	args := m.Called(ctx, call)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Call), args.Error(1)
}
func (m *mockCallRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.Call, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Call), args.Error(1)
}
func (m *mockCallRepo) List(ctx context.Context, filter ports.ListCallsFilter, page, perPage int) ([]*entities.Call, int, error) {
	args := m.Called(ctx, filter, page, perPage)
	return args.Get(0).([]*entities.Call), args.Int(1), args.Error(2)
}
func (m *mockCallRepo) Update(ctx context.Context, call *entities.Call) (*entities.Call, error) {
	args := m.Called(ctx, call)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Call), args.Error(1)
}
func (m *mockCallRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// ─── Helper ───────────────────────────────────────────────────────────────────

func newTestCallService(repo *mockCallRepo) *services.CallService {
	return services.NewCallService(repo, zap.NewNop())
}

func pendingCall(userID uuid.UUID) *entities.Call {
	return &entities.Call{
		ID:        uuid.New(),
		UserID:    userID,
		Title:     "Sales Call Q1",
		Status:    "pending",
		Metadata:  map[string]interface{}{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ─── CreateCall ───────────────────────────────────────────────────────────────

func TestCreateCall_Success(t *testing.T) {
	repo := new(mockCallRepo)
	svc := newTestCallService(repo)
	ctx := context.Background()

	userID := uuid.New()
	expected := pendingCall(userID)

	repo.On("Create", ctx, mock.AnythingOfType("*entities.Call")).Return(expected, nil)

	call, err := svc.CreateCall(ctx, services.CreateCallInput{UserID: userID, Title: "Sales Call Q1"})

	require.NoError(t, err)
	assert.Equal(t, expected.ID, call.ID)
	assert.Equal(t, "pending", call.Status)
}

func TestCreateCall_MissingTitle(t *testing.T) {
	svc := newTestCallService(new(mockCallRepo))

	_, err := svc.CreateCall(context.Background(), services.CreateCallInput{UserID: uuid.New(), Title: ""})

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 400, appErr.Status)
}

func TestCreateCall_MissingUserID(t *testing.T) {
	svc := newTestCallService(new(mockCallRepo))

	_, err := svc.CreateCall(context.Background(), services.CreateCallInput{UserID: uuid.Nil, Title: "Valid Title"})

	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 400, appErr.Status)
}

func TestCreateCall_NilMetadataDefaultsToEmpty(t *testing.T) {
	repo := new(mockCallRepo)
	svc := newTestCallService(repo)
	ctx := context.Background()

	userID := uuid.New()
	repo.On("Create", ctx, mock.MatchedBy(func(c *entities.Call) bool {
		return c.Metadata != nil // metadata must be initialised to an empty map
	})).Return(pendingCall(userID), nil)

	_, err := svc.CreateCall(ctx, services.CreateCallInput{UserID: userID, Title: "No Metadata", Metadata: nil})

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestCreateCall_RepoCreateError(t *testing.T) {
	repo := new(mockCallRepo)
	svc := newTestCallService(repo)
	ctx := context.Background()

	userID := uuid.New()
	repo.On("Create", ctx, mock.AnythingOfType("*entities.Call")).Return(nil, assert.AnError)

	_, err := svc.CreateCall(ctx, services.CreateCallInput{UserID: userID, Title: "X"})
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

// ─── GetCall ──────────────────────────────────────────────────────────────────

func TestGetCall_Success(t *testing.T) {
	repo := new(mockCallRepo)
	svc := newTestCallService(repo)
	ctx := context.Background()

	call := pendingCall(uuid.New())
	repo.On("GetByID", ctx, call.ID).Return(call, nil)

	result, err := svc.GetCall(ctx, call.ID)

	require.NoError(t, err)
	assert.Equal(t, call.ID, result.ID)
}

func TestGetCall_NotFound(t *testing.T) {
	repo := new(mockCallRepo)
	svc := newTestCallService(repo)
	ctx := context.Background()
	id := uuid.New()

	repo.On("GetByID", ctx, id).Return(nil, apperr.NotFound("Call", id.String()))

	_, err := svc.GetCall(ctx, id)

	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestGetCall_RepoError(t *testing.T) {
	repo := new(mockCallRepo)
	svc := newTestCallService(repo)
	ctx := context.Background()
	id := uuid.New()

	repo.On("GetByID", ctx, id).Return(nil, assert.AnError)

	_, err := svc.GetCall(ctx, id)
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, 500, appErr.Status)
}

// ─── ListCalls ────────────────────────────────────────────────────────────────

func TestListCalls_ReturnsPage(t *testing.T) {
	repo := new(mockCallRepo)
	svc := newTestCallService(repo)
	ctx := context.Background()

	userID := uuid.New()
	filter := ports.ListCallsFilter{UserID: &userID}
	calls := []*entities.Call{pendingCall(userID), pendingCall(userID)}

	repo.On("List", ctx, filter, 1, 20).Return(calls, 2, nil)

	result, total, err := svc.ListCalls(ctx, filter, 1, 20)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 2, total)
}

func TestListCalls_RepoError(t *testing.T) {
	repo := new(mockCallRepo)
	svc := newTestCallService(repo)
	ctx := context.Background()

	filter := ports.ListCallsFilter{}
	repo.On("List", ctx, filter, 1, 20).Return([]*entities.Call(nil), 0, assert.AnError)

	_, _, err := svc.ListCalls(ctx, filter, 1, 20)
	require.Error(t, err)
}

func TestListCalls_EmptyResult(t *testing.T) {
	repo := new(mockCallRepo)
	svc := newTestCallService(repo)
	ctx := context.Background()

	filter := ports.ListCallsFilter{}
	repo.On("List", ctx, filter, 1, 20).Return([]*entities.Call{}, 0, nil)

	result, total, err := svc.ListCalls(ctx, filter, 1, 20)

	require.NoError(t, err)
	assert.Empty(t, result)
	assert.Equal(t, 0, total)
}

// ─── UpdateCall ───────────────────────────────────────────────────────────────

func TestUpdateCall_PartialUpdate(t *testing.T) {
	repo := new(mockCallRepo)
	svc := newTestCallService(repo)
	ctx := context.Background()

	call := pendingCall(uuid.New())
	newTitle := "Updated Title"
	newStatus := "processing"
	updated := *call
	updated.Title = newTitle
	updated.Status = newStatus

	repo.On("GetByID", ctx, call.ID).Return(call, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*entities.Call")).Return(&updated, nil)

	result, err := svc.UpdateCall(ctx, call.ID, services.UpdateCallInput{
		Title:  &newTitle,
		Status: &newStatus,
	})

	require.NoError(t, err)
	assert.Equal(t, "Updated Title", result.Title)
	assert.Equal(t, "processing", result.Status)
}

func TestUpdateCall_DurationCalculated(t *testing.T) {
	repo := new(mockCallRepo)
	svc := newTestCallService(repo)
	ctx := context.Background()

	start := time.Now().Add(-30 * time.Minute)
	end := time.Now()
	call := pendingCall(uuid.New())
	call.StartedAt = &start

	repo.On("GetByID", ctx, call.ID).Return(call, nil)
	repo.On("Update", ctx, mock.MatchedBy(func(c *entities.Call) bool {
		return c.DurationSec == 1800 // 30 min in seconds
	})).Return(call, nil)

	_, err := svc.UpdateCall(ctx, call.ID, services.UpdateCallInput{EndedAt: &end})

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUpdateCall_NotFound(t *testing.T) {
	repo := new(mockCallRepo)
	svc := newTestCallService(repo)
	ctx := context.Background()
	id := uuid.New()

	repo.On("GetByID", ctx, id).Return(nil, apperr.NotFound("Call", id.String()))

	newTitle := "X"
	_, err := svc.UpdateCall(ctx, id, services.UpdateCallInput{Title: &newTitle})

	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestUpdateCall_AllFields(t *testing.T) {
	// Exercises every optional input field branch in UpdateCall.
	repo := new(mockCallRepo)
	svc := newTestCallService(repo)
	ctx := context.Background()

	start := time.Now().Add(-10 * time.Minute)
	call := pendingCall(uuid.New())
	call.StartedAt = &start
	repo.On("GetByID", ctx, call.ID).Return(call, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*entities.Call")).Return(call, nil)

	end := time.Now()
	title := "Retitled"
	status := "done"
	rec := "s3://bucket/rec.mp4"
	transcript := "Hello world"
	newStart := time.Now().Add(-15 * time.Minute)
	meta := entities.JSONMap{"tag": "demo"}

	_, err := svc.UpdateCall(ctx, call.ID, services.UpdateCallInput{
		Title:        &title,
		Status:       &status,
		RecordingURL: &rec,
		Transcript:   &transcript,
		StartedAt:    &newStart,
		EndedAt:      &end,
		Metadata:     meta,
	})
	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUpdateCall_RepoUpdateError(t *testing.T) {
	repo := new(mockCallRepo)
	svc := newTestCallService(repo)
	ctx := context.Background()

	call := pendingCall(uuid.New())
	repo.On("GetByID", ctx, call.ID).Return(call, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*entities.Call")).Return(nil, assert.AnError)

	title := "X"
	_, err := svc.UpdateCall(ctx, call.ID, services.UpdateCallInput{Title: &title})
	require.Error(t, err)
}

// ─── DeleteCall ───────────────────────────────────────────────────────────────

func TestDeleteCall_Success(t *testing.T) {
	repo := new(mockCallRepo)
	svc := newTestCallService(repo)
	ctx := context.Background()

	call := pendingCall(uuid.New())
	repo.On("GetByID", ctx, call.ID).Return(call, nil)
	repo.On("SoftDelete", ctx, call.ID).Return(nil)

	require.NoError(t, svc.DeleteCall(ctx, call.ID))
	repo.AssertExpectations(t)
}

func TestDeleteCall_NotFound(t *testing.T) {
	repo := new(mockCallRepo)
	svc := newTestCallService(repo)
	ctx := context.Background()
	id := uuid.New()

	repo.On("GetByID", ctx, id).Return(nil, apperr.NotFound("Call", id.String()))

	err := svc.DeleteCall(ctx, id)

	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestDeleteCall_RepoSoftDeleteError(t *testing.T) {
	repo := new(mockCallRepo)
	svc := newTestCallService(repo)
	ctx := context.Background()

	call := pendingCall(uuid.New())
	repo.On("GetByID", ctx, call.ID).Return(call, nil)
	repo.On("SoftDelete", ctx, call.ID).Return(assert.AnError)

	err := svc.DeleteCall(ctx, call.ID)
	require.Error(t, err)
}
