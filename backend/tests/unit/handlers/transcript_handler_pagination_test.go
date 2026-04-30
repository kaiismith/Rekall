package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/domain/ports"
	"github.com/rekall/backend/internal/interfaces/http/dto"
	"github.com/rekall/backend/internal/interfaces/http/handlers"
)

// captureTranscriptRepo wraps fakeTranscriptRepoH but lets a test pre-set the
// values returned by ListSegmentsByMeeting / ListSegmentsByCall and capture
// the pagination args the handler passed in.
type captureTranscriptRepo struct {
	*fakeTranscriptRepoH
	calledPage    int
	calledPerPage int
	cannedSegs    []*entities.TranscriptSegment
	cannedTotal   int
}

func newCaptureRepo(total int, segs []*entities.TranscriptSegment) *captureTranscriptRepo {
	return &captureTranscriptRepo{
		fakeTranscriptRepoH: newFakeTranscriptRepoH(),
		cannedSegs:          segs,
		cannedTotal:         total,
	}
}

func (c *captureTranscriptRepo) ListSegmentsByMeeting(_ context.Context, _ uuid.UUID, page, perPage int) ([]*entities.TranscriptSegment, int, error) {
	c.calledPage = page
	c.calledPerPage = perPage
	return c.cannedSegs, c.cannedTotal, nil
}

func (c *captureTranscriptRepo) ListSegmentsByCall(_ context.Context, _ uuid.UUID, page, perPage int) ([]*entities.TranscriptSegment, int, error) {
	c.calledPage = page
	c.calledPerPage = perPage
	return c.cannedSegs, c.cannedTotal, nil
}

var _ ports.TranscriptRepository = (*captureTranscriptRepo)(nil)

// helper — build a router for the meeting transcript endpoint with a given caller.
func meetingRouterFor(h *handlers.TranscriptHandler, callerID uuid.UUID) *gin.Engine {
	r := gin.New()
	r.Use(injectClaims(callerID, "member"))
	r.GET("/api/v1/meetings/:code/transcript", h.GetMeetingTranscript)
	return r
}

func TestGetMeetingTranscript_Pagination_HappyPath(t *testing.T) {
	host := uuid.New()
	meetingID := uuid.New()

	// 3 canned segments returned for page 2 of a total-of-25 result.
	canned := []*entities.TranscriptSegment{
		{ID: uuid.New(), SegmentIndex: 10, SpeakerUserID: host, MeetingID: &meetingID,
			Text: "ten", StartMs: 0, EndMs: 100, EngineMode: "openai", ModelID: "whisper-1",
			SegmentStartedAt: time.Now()},
	}
	tr := newCaptureRepo(25, canned)
	cr := new(mockCallRepo)
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	mr.On("GetByCode", mock.Anything, "abc-defg-hij").Return(&entities.Meeting{
		ID: meetingID, HostID: host,
	}, nil)

	h := handlers.NewTranscriptHandler(tr, cr, mr, pr, zap.NewNop())
	router := meetingRouterFor(h, host)

	w := doRequest(router, http.MethodGet, "/api/v1/meetings/abc-defg-hij/transcript?page=2&per_page=10", nil)
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	assert.Equal(t, 2, tr.calledPage, "page param forwarded to repo")
	assert.Equal(t, 10, tr.calledPerPage, "per_page param forwarded to repo")

	var env dto.MeetingTranscriptEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, 2, env.Data.Pagination.Page)
	assert.Equal(t, 10, env.Data.Pagination.PerPage)
	assert.Equal(t, 25, env.Data.Pagination.Total)
	assert.Equal(t, 3, env.Data.Pagination.TotalPages)
	assert.True(t, env.Data.Pagination.HasMore, "page 2 of 3 has more")
}

func TestGetMeetingTranscript_Pagination_DefaultsWhenMissing(t *testing.T) {
	host := uuid.New()
	meetingID := uuid.New()
	tr := newCaptureRepo(0, nil)
	cr := new(mockCallRepo)
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	mr.On("GetByCode", mock.Anything, "abc-defg-hij").Return(&entities.Meeting{
		ID: meetingID, HostID: host,
	}, nil)

	h := handlers.NewTranscriptHandler(tr, cr, mr, pr, zap.NewNop())
	router := meetingRouterFor(h, host)

	w := doRequest(router, http.MethodGet, "/api/v1/meetings/abc-defg-hij/transcript", nil)
	require.Equal(t, http.StatusOK, w.Code)

	assert.Equal(t, 1, tr.calledPage)
	assert.Equal(t, 50, tr.calledPerPage)

	var env dto.MeetingTranscriptEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, 1, env.Data.Pagination.Page)
	assert.Equal(t, 50, env.Data.Pagination.PerPage)
	assert.Equal(t, 0, env.Data.Pagination.Total)
	assert.Equal(t, 0, env.Data.Pagination.TotalPages)
	assert.False(t, env.Data.Pagination.HasMore, "empty meeting → no more pages")
}

func TestGetMeetingTranscript_Pagination_Clamps(t *testing.T) {
	cases := []struct {
		name       string
		query      string
		wantPage   int
		wantPerPag int
	}{
		{"per_page above max clamps to 200", "?page=1&per_page=99999", 1, 200},
		{"per_page=0 falls back to default", "?page=1&per_page=0", 1, 50},
		{"per_page non-int falls back to default", "?page=1&per_page=abc", 1, 50},
		{"per_page negative falls back to default", "?page=1&per_page=-3", 1, 50},
		{"page non-int falls back to 1", "?page=abc&per_page=20", 1, 20},
		{"page=0 falls back to 1", "?page=0&per_page=20", 1, 20},
		{"page negative falls back to 1", "?page=-2&per_page=20", 1, 20},
	}

	host := uuid.New()
	meetingID := uuid.New()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tr := newCaptureRepo(0, nil)
			cr := new(mockCallRepo)
			mr := new(mockMeetingRepo)
			pr := new(mockParticipantRepo)
			mr.On("GetByCode", mock.Anything, "abc-defg-hij").Return(&entities.Meeting{
				ID: meetingID, HostID: host,
			}, nil)

			h := handlers.NewTranscriptHandler(tr, cr, mr, pr, zap.NewNop())
			router := meetingRouterFor(h, host)

			w := doRequest(router, http.MethodGet, "/api/v1/meetings/abc-defg-hij/transcript"+tc.query, nil)
			require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

			assert.Equal(t, tc.wantPage, tr.calledPage, "page clamping")
			assert.Equal(t, tc.wantPerPag, tr.calledPerPage, "per_page clamping")
		})
	}
}

func TestGetMeetingTranscript_LastPage_HasMoreFalse(t *testing.T) {
	host := uuid.New()
	meetingID := uuid.New()
	// total=75, per_page=50 → total_pages=2; page=2 ⇒ has_more=false
	tr := newCaptureRepo(75, nil)
	cr := new(mockCallRepo)
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	mr.On("GetByCode", mock.Anything, "abc-defg-hij").Return(&entities.Meeting{
		ID: meetingID, HostID: host,
	}, nil)

	h := handlers.NewTranscriptHandler(tr, cr, mr, pr, zap.NewNop())
	router := meetingRouterFor(h, host)

	q := url.Values{"page": {"2"}, "per_page": {"50"}}
	w := doRequest(router, http.MethodGet, "/api/v1/meetings/abc-defg-hij/transcript?"+q.Encode(), nil)
	require.Equal(t, http.StatusOK, w.Code)

	var env dto.MeetingTranscriptEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, 2, env.Data.Pagination.TotalPages)
	assert.False(t, env.Data.Pagination.HasMore, "last page must have_more=false")
}

func TestGetMeetingTranscript_403_Logs_NotHostNotParticipant(t *testing.T) {
	// This re-asserts the 403 path still returns the right code after the
	// handler now also emits a structured warn log on denial.
	tr := newCaptureRepo(0, nil)
	cr := new(mockCallRepo)
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)

	host := uuid.New()
	intruder := uuid.New()
	meetingID := uuid.New()
	mr.On("GetByCode", mock.Anything, "abc-defg-hij").Return(&entities.Meeting{
		ID: meetingID, HostID: host,
	}, nil)
	pr.On("GetByMeetingAndUser", mock.Anything, meetingID, intruder).Return(nil, nil)

	h := handlers.NewTranscriptHandler(tr, cr, mr, pr, zap.NewNop())
	router := meetingRouterFor(h, intruder)

	w := doRequest(router, http.MethodGet, "/api/v1/meetings/abc-defg-hij/transcript", nil)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ─── Symmetry: call transcript pagination ─────────────────────────────────────

func callRouterFor(h *handlers.TranscriptHandler, callerID uuid.UUID) *gin.Engine {
	r := gin.New()
	r.Use(injectClaims(callerID, "member"))
	r.GET("/api/v1/calls/:id/transcript", h.GetCallTranscript)
	return r
}

func TestGetCallTranscript_Pagination_HappyPath(t *testing.T) {
	owner := uuid.New()
	callID := uuid.New()
	tr := newCaptureRepo(12, nil)
	cr := new(mockCallRepo)
	mr := new(mockMeetingRepo)
	pr := new(mockParticipantRepo)
	cr.On("GetByID", mock.Anything, callID).Return(&entities.Call{
		ID: callID, UserID: owner,
	}, nil)

	h := handlers.NewTranscriptHandler(tr, cr, mr, pr, zap.NewNop())
	router := callRouterFor(h, owner)

	w := doRequest(router, http.MethodGet, "/api/v1/calls/"+callID.String()+"/transcript?page=3&per_page=4", nil)
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	assert.Equal(t, 3, tr.calledPage)
	assert.Equal(t, 4, tr.calledPerPage)

	var env dto.CallTranscriptEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, 3, env.Data.Pagination.Page)
	assert.Equal(t, 4, env.Data.Pagination.PerPage)
	assert.Equal(t, 12, env.Data.Pagination.Total)
	assert.Equal(t, 3, env.Data.Pagination.TotalPages, "ceil(12/4) = 3")
	assert.False(t, env.Data.Pagination.HasMore, "last page")
}
