package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	"github.com/rekall/backend/internal/infrastructure/storage"
	"github.com/rekall/backend/internal/interfaces/http/dto"
	"github.com/rekall/backend/internal/interfaces/http/handlers"
	wsHub "github.com/rekall/backend/internal/interfaces/http/ws"
)

// fakeSpeakerTranscriptRepo wraps fakeTranscriptRepoH but lets the test
// override ListSpeakerUserIDsByMeeting with canned ids.
type fakeSpeakerTranscriptRepo struct {
	*fakeTranscriptRepoH
	speakerIDs []uuid.UUID
}

func (f *fakeSpeakerTranscriptRepo) ListSpeakerUserIDsByMeeting(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, len(f.speakerIDs))
	copy(out, f.speakerIDs)
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out, nil
}

var _ ports.TranscriptRepository = (*fakeSpeakerTranscriptRepo)(nil)

// buildHandlerWithSpeakers wires a MeetingHandler with a fake transcript repo
// + mock user repo so the GetByCode handler can populate Speakers.
func buildHandlerWithSpeakers(
	mr *mockMeetingRepo,
	pr *mockParticipantRepo,
	tr ports.TranscriptRepository,
	ur ports.UserRepository,
) *handlers.MeetingHandler {
	svc := newMeetingService(mr, pr)
	manager := wsHub.NewHubManager(nil, nil, zap.NewNop())
	store := storage.NewMemoryWSTicketStore(zap.NewNop())
	return handlers.NewMeetingHandler(
		svc, nil, nil, tr, ur, manager, store, "http://rekall.test", zap.NewNop(),
	)
}

func newRouterWithCaller(h *handlers.MeetingHandler, callerID uuid.UUID) *gin.Engine {
	r := gin.New()
	r.Use(injectClaims(callerID, "member"))
	r.GET("/meetings/:code", h.GetByCode)
	return r
}

func TestMeetingGetByCode_SpeakersPopulated(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	host := uuid.New()
	meeting := activeMeeting(host)
	mr.On("GetByCode", mock.Anything, "abc-defg-hij").Return(meeting, nil)

	speakerA := uuid.New()
	speakerB := uuid.New()
	tr := &fakeSpeakerTranscriptRepo{
		fakeTranscriptRepoH: newFakeTranscriptRepoH(),
		speakerIDs:          []uuid.UUID{speakerA, speakerB},
	}
	ur := new(mockUserRepo)
	// FindByIDs is invoked with the (already-sorted) slice the repo returned;
	// be flexible about argument order to keep the test stable.
	ur.On("FindByIDs", mock.Anything, mock.AnythingOfType("[]uuid.UUID")).Return(
		[]*entities.User{
			{ID: speakerA, FullName: "Alice Nguyen"},
			{ID: speakerB, FullName: "Bob Carter"},
		}, nil,
	)

	h := buildHandlerWithSpeakers(mr, pr, tr, ur)
	r := newRouterWithCaller(h, host)

	w := doRequest(r, http.MethodGet, "/meetings/abc-defg-hij", nil)
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	var env dto.MeetingResponseEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	require.Len(t, env.Data.Speakers, 2)

	byID := map[string]dto.SpeakerInfo{}
	for _, s := range env.Data.Speakers {
		byID[s.UserID] = s
	}
	assert.Equal(t, "Alice Nguyen", byID[speakerA.String()].FullName)
	assert.Equal(t, "AN", byID[speakerA.String()].Initials)
	assert.Equal(t, "Bob Carter", byID[speakerB.String()].FullName)
	assert.Equal(t, "BC", byID[speakerB.String()].Initials)
}

func TestMeetingGetByCode_SpeakersEmptyWhenNoSessions(t *testing.T) {
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	host := uuid.New()
	meeting := activeMeeting(host)
	mr.On("GetByCode", mock.Anything, "abc-defg-hij").Return(meeting, nil)

	tr := &fakeSpeakerTranscriptRepo{
		fakeTranscriptRepoH: newFakeTranscriptRepoH(),
		speakerIDs:          []uuid.UUID{},
	}
	ur := new(mockUserRepo)

	h := buildHandlerWithSpeakers(mr, pr, tr, ur)
	r := newRouterWithCaller(h, host)

	w := doRequest(r, http.MethodGet, "/meetings/abc-defg-hij", nil)
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	var env dto.MeetingResponseEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.NotNil(t, env.Data.Speakers, "speakers must be [] not null")
	assert.Empty(t, env.Data.Speakers)
	ur.AssertNotCalled(t, "FindByIDs", mock.Anything, mock.Anything)
}
