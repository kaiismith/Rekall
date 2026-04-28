package httphelpers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rekall/backend/internal/domain/ports"
	"github.com/rekall/backend/internal/interfaces/http/dto"
	apperr "github.com/rekall/backend/pkg/errors"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func ginCtxWithQuery(query string) *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/?"+query, nil)
	c.Request = req
	return c
}

func TestParseScopeFilter_NoParams_ReturnsNil(t *testing.T) {
	scope, err := dto.ParseScopeFilter(ginCtxWithQuery(""))
	require.NoError(t, err)
	assert.Nil(t, scope)
}

func TestParseScopeFilter_OpenScope(t *testing.T) {
	scope, err := dto.ParseScopeFilter(ginCtxWithQuery("filter[scope_type]=open"))
	require.NoError(t, err)
	require.NotNil(t, scope)
	assert.Equal(t, ports.ScopeKindOpen, scope.Kind)
	assert.Equal(t, uuid.Nil, scope.ID)
}

func TestParseScopeFilter_OpenWithIDRejected(t *testing.T) {
	scope, err := dto.ParseScopeFilter(ginCtxWithQuery("filter[scope_type]=open&filter[scope_id]=" + uuid.NewString()))
	require.Error(t, err)
	assert.Nil(t, scope)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, appErr.Status)
}

func TestParseScopeFilter_OrgScope_OK(t *testing.T) {
	id := uuid.New()
	scope, err := dto.ParseScopeFilter(ginCtxWithQuery("filter[scope_type]=organization&filter[scope_id]=" + id.String()))
	require.NoError(t, err)
	require.NotNil(t, scope)
	assert.Equal(t, ports.ScopeKindOrganization, scope.Kind)
	assert.Equal(t, id, scope.ID)
}

func TestParseScopeFilter_DeptScope_OK(t *testing.T) {
	id := uuid.New()
	scope, err := dto.ParseScopeFilter(ginCtxWithQuery("filter[scope_type]=department&filter[scope_id]=" + id.String()))
	require.NoError(t, err)
	require.NotNil(t, scope)
	assert.Equal(t, ports.ScopeKindDepartment, scope.Kind)
	assert.Equal(t, id, scope.ID)
}

func TestParseScopeFilter_OrgWithoutID_Rejected(t *testing.T) {
	_, err := dto.ParseScopeFilter(ginCtxWithQuery("filter[scope_type]=organization"))
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, appErr.Status)
}

func TestParseScopeFilter_OrgWithBadUUID_Rejected(t *testing.T) {
	_, err := dto.ParseScopeFilter(ginCtxWithQuery("filter[scope_type]=organization&filter[scope_id]=not-a-uuid"))
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, appErr.Status)
}

func TestParseScopeFilter_IDWithoutType_Rejected(t *testing.T) {
	_, err := dto.ParseScopeFilter(ginCtxWithQuery("filter[scope_id]=" + uuid.NewString()))
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, appErr.Status)
}

func TestParseScopeFilter_UnknownType_Rejected(t *testing.T) {
	_, err := dto.ParseScopeFilter(ginCtxWithQuery("filter[scope_type]=other"))
	require.Error(t, err)
	appErr, ok := apperr.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, appErr.Status)
}
