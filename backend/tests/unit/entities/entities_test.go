package entities_test

import (
	"database/sql/driver"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/pkg/constants"
)

// ─── TableName methods ─────────────────────────────────────────────────────────

func TestTableNames(t *testing.T) {
	assert.Equal(t, "users", entities.User{}.TableName())
	assert.Equal(t, "organizations", entities.Organization{}.TableName())
	assert.Equal(t, "org_memberships", entities.OrgMembership{}.TableName())
	assert.Equal(t, "departments", entities.Department{}.TableName())
	assert.Equal(t, "department_memberships", entities.DepartmentMembership{}.TableName())
	assert.Equal(t, "meetings", entities.Meeting{}.TableName())
	assert.Equal(t, "meeting_participants", entities.MeetingParticipant{}.TableName())
	assert.Equal(t, "invitations", entities.Invitation{}.TableName())
	assert.Equal(t, "refresh_tokens", entities.RefreshToken{}.TableName())
	assert.Equal(t, "email_verification_tokens", entities.EmailVerificationToken{}.TableName())
	assert.Equal(t, "password_reset_tokens", entities.PasswordResetToken{}.TableName())
	assert.Equal(t, "calls", entities.Call{}.TableName())
}

// ─── User ──────────────────────────────────────────────────────────────────────

func TestUser_IsAdmin(t *testing.T) {
	assert.True(t, (&entities.User{Role: "admin"}).IsAdmin())
	assert.False(t, (&entities.User{Role: "member"}).IsAdmin())
	assert.False(t, (&entities.User{Role: ""}).IsAdmin())
}

func TestUser_IsDeleted(t *testing.T) {
	now := time.Now()
	assert.True(t, (&entities.User{DeletedAt: &now}).IsDeleted())
	assert.False(t, (&entities.User{DeletedAt: nil}).IsDeleted())
}

func TestUser_IsEmailVerified(t *testing.T) {
	assert.True(t, (&entities.User{EmailVerified: true}).IsEmailVerified())
	assert.False(t, (&entities.User{EmailVerified: false}).IsEmailVerified())
}

// ─── Organization ──────────────────────────────────────────────────────────────

func TestOrganization_IsDeleted(t *testing.T) {
	now := time.Now()
	assert.True(t, (&entities.Organization{DeletedAt: &now}).IsDeleted())
	assert.False(t, (&entities.Organization{DeletedAt: nil}).IsDeleted())
}

// ─── OrgMembership ─────────────────────────────────────────────────────────────

func TestOrgMembership_IsOwner(t *testing.T) {
	assert.True(t, (&entities.OrgMembership{Role: constants.OrgRoleOwner}).IsOwner())
	assert.False(t, (&entities.OrgMembership{Role: constants.OrgRoleAdmin}).IsOwner())
	assert.False(t, (&entities.OrgMembership{Role: constants.OrgRoleMember}).IsOwner())
}

func TestOrgMembership_IsAdmin(t *testing.T) {
	// IsAdmin is true for both owner and admin roles.
	assert.True(t, (&entities.OrgMembership{Role: constants.OrgRoleOwner}).IsAdmin())
	assert.True(t, (&entities.OrgMembership{Role: constants.OrgRoleAdmin}).IsAdmin())
	assert.False(t, (&entities.OrgMembership{Role: constants.OrgRoleMember}).IsAdmin())
}

func TestOrgMembership_CanManageMembers(t *testing.T) {
	assert.True(t, (&entities.OrgMembership{Role: constants.OrgRoleOwner}).CanManageMembers())
	assert.True(t, (&entities.OrgMembership{Role: constants.OrgRoleAdmin}).CanManageMembers())
	assert.False(t, (&entities.OrgMembership{Role: constants.OrgRoleMember}).CanManageMembers())
}

// ─── Department / DepartmentMembership ─────────────────────────────────────────

func TestDepartment_IsDeleted(t *testing.T) {
	now := time.Now()
	assert.True(t, (&entities.Department{DeletedAt: &now}).IsDeleted())
	assert.False(t, (&entities.Department{DeletedAt: nil}).IsDeleted())
}

func TestDepartmentMembership_IsHead(t *testing.T) {
	assert.True(t, (&entities.DepartmentMembership{Role: constants.DeptRoleHead}).IsHead())
	assert.False(t, (&entities.DepartmentMembership{Role: "member"}).IsHead())
}

// ─── Meeting ───────────────────────────────────────────────────────────────────

func TestMeeting_StatusChecks(t *testing.T) {
	assert.True(t, (&entities.Meeting{Status: entities.MeetingStatusActive}).IsActive())
	assert.False(t, (&entities.Meeting{Status: entities.MeetingStatusEnded}).IsActive())

	assert.True(t, (&entities.Meeting{Status: entities.MeetingStatusEnded}).IsEnded())
	assert.False(t, (&entities.Meeting{Status: entities.MeetingStatusActive}).IsEnded())

	assert.True(t, (&entities.Meeting{Status: entities.MeetingStatusWaiting}).IsWaiting())
	assert.False(t, (&entities.Meeting{Status: entities.MeetingStatusActive}).IsWaiting())
}

func TestMeeting_IsPrivate(t *testing.T) {
	assert.True(t, (&entities.Meeting{Type: entities.MeetingTypePrivate}).IsPrivate())
	assert.False(t, (&entities.Meeting{Type: entities.MeetingTypeOpen}).IsPrivate())
}

func TestMeeting_JoinURL(t *testing.T) {
	m := &entities.Meeting{Code: "abc-123"}
	assert.Equal(t, "https://example.com/meeting/abc-123", m.JoinURL("https://example.com"))
	assert.Equal(t, "/meeting/abc-123", m.JoinURL(""))
}

// ─── MeetingParticipant ────────────────────────────────────────────────────────

func TestMeetingParticipant_IsActive(t *testing.T) {
	now := time.Now()

	// Joined, not left → active
	assert.True(t, (&entities.MeetingParticipant{JoinedAt: &now, LeftAt: nil}).IsActive())

	// Joined and left → not active
	assert.False(t, (&entities.MeetingParticipant{JoinedAt: &now, LeftAt: &now}).IsActive())

	// Not joined yet → not active
	assert.False(t, (&entities.MeetingParticipant{JoinedAt: nil, LeftAt: nil}).IsActive())
}

func TestMeetingParticipant_IsHost(t *testing.T) {
	assert.True(t, (&entities.MeetingParticipant{Role: entities.ParticipantRoleHost}).IsHost())
	assert.False(t, (&entities.MeetingParticipant{Role: entities.ParticipantRoleParticipant}).IsHost())
}

// ─── Invitation ────────────────────────────────────────────────────────────────

func TestInvitation_IsExpired(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	future := time.Now().Add(1 * time.Hour)

	assert.True(t, (&entities.Invitation{ExpiresAt: past}).IsExpired())
	assert.False(t, (&entities.Invitation{ExpiresAt: future}).IsExpired())
}

func TestInvitation_IsAccepted(t *testing.T) {
	now := time.Now()
	assert.True(t, (&entities.Invitation{AcceptedAt: &now}).IsAccepted())
	assert.False(t, (&entities.Invitation{AcceptedAt: nil}).IsAccepted())
}

func TestInvitation_IsValid(t *testing.T) {
	future := time.Now().Add(1 * time.Hour)
	past := time.Now().Add(-1 * time.Hour)
	now := time.Now()

	// Future expiry + not accepted → valid
	assert.True(t, (&entities.Invitation{ExpiresAt: future, AcceptedAt: nil}).IsValid())
	// Past expiry → not valid
	assert.False(t, (&entities.Invitation{ExpiresAt: past, AcceptedAt: nil}).IsValid())
	// Already accepted → not valid
	assert.False(t, (&entities.Invitation{ExpiresAt: future, AcceptedAt: &now}).IsValid())
}

// ─── RefreshToken ──────────────────────────────────────────────────────────────

func TestRefreshToken_IsExpired(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	future := time.Now().Add(1 * time.Hour)

	assert.True(t, (&entities.RefreshToken{ExpiresAt: past}).IsExpired())
	assert.False(t, (&entities.RefreshToken{ExpiresAt: future}).IsExpired())
}

func TestRefreshToken_IsRevoked(t *testing.T) {
	now := time.Now()
	assert.True(t, (&entities.RefreshToken{RevokedAt: &now}).IsRevoked())
	assert.False(t, (&entities.RefreshToken{RevokedAt: nil}).IsRevoked())
}

func TestRefreshToken_IsValid(t *testing.T) {
	future := time.Now().Add(1 * time.Hour)
	past := time.Now().Add(-1 * time.Hour)
	now := time.Now()

	assert.True(t, (&entities.RefreshToken{ExpiresAt: future, RevokedAt: nil}).IsValid())
	assert.False(t, (&entities.RefreshToken{ExpiresAt: past, RevokedAt: nil}).IsValid())
	assert.False(t, (&entities.RefreshToken{ExpiresAt: future, RevokedAt: &now}).IsValid())
}

// ─── EmailVerificationToken ────────────────────────────────────────────────────

func TestEmailVerificationToken_Methods(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	future := time.Now().Add(1 * time.Hour)
	now := time.Now()

	assert.True(t, (&entities.EmailVerificationToken{ExpiresAt: past}).IsExpired())
	assert.False(t, (&entities.EmailVerificationToken{ExpiresAt: future}).IsExpired())

	assert.True(t, (&entities.EmailVerificationToken{UsedAt: &now}).IsUsed())
	assert.False(t, (&entities.EmailVerificationToken{UsedAt: nil}).IsUsed())

	assert.True(t, (&entities.EmailVerificationToken{ExpiresAt: future, UsedAt: nil}).IsValid())
	assert.False(t, (&entities.EmailVerificationToken{ExpiresAt: past, UsedAt: nil}).IsValid())
	assert.False(t, (&entities.EmailVerificationToken{ExpiresAt: future, UsedAt: &now}).IsValid())
}

// ─── PasswordResetToken ────────────────────────────────────────────────────────

func TestPasswordResetToken_Methods(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	future := time.Now().Add(1 * time.Hour)
	now := time.Now()

	assert.True(t, (&entities.PasswordResetToken{ExpiresAt: past}).IsExpired())
	assert.False(t, (&entities.PasswordResetToken{ExpiresAt: future}).IsExpired())

	assert.True(t, (&entities.PasswordResetToken{UsedAt: &now}).IsUsed())
	assert.False(t, (&entities.PasswordResetToken{UsedAt: nil}).IsUsed())

	assert.True(t, (&entities.PasswordResetToken{ExpiresAt: future, UsedAt: nil}).IsValid())
	assert.False(t, (&entities.PasswordResetToken{ExpiresAt: past, UsedAt: nil}).IsValid())
	assert.False(t, (&entities.PasswordResetToken{ExpiresAt: future, UsedAt: &now}).IsValid())
}

// ─── Call ──────────────────────────────────────────────────────────────────────

func TestCall_StatusChecks(t *testing.T) {
	assert.True(t, (&entities.Call{Status: "pending"}).IsPending())
	assert.False(t, (&entities.Call{Status: "done"}).IsPending())

	assert.True(t, (&entities.Call{Status: "processing"}).IsProcessing())
	assert.False(t, (&entities.Call{Status: "pending"}).IsProcessing())

	assert.True(t, (&entities.Call{Status: "done"}).IsDone())
	assert.False(t, (&entities.Call{Status: "failed"}).IsDone())

	assert.True(t, (&entities.Call{Status: "failed"}).IsFailed())
	assert.False(t, (&entities.Call{Status: "done"}).IsFailed())
}

func TestCall_IsDeleted(t *testing.T) {
	now := time.Now()
	assert.True(t, (&entities.Call{DeletedAt: &now}).IsDeleted())
	assert.False(t, (&entities.Call{DeletedAt: nil}).IsDeleted())
}

// ─── JSONMap ───────────────────────────────────────────────────────────────────

func TestJSONMap_Value_NilMap(t *testing.T) {
	var j entities.JSONMap
	v, err := j.Value()
	require.NoError(t, err)
	assert.Equal(t, "{}", v)
}

func TestJSONMap_Value_EmptyMap(t *testing.T) {
	j := entities.JSONMap{}
	v, err := j.Value()
	require.NoError(t, err)
	assert.Equal(t, "{}", v)
}

func TestJSONMap_Value_PopulatedMap(t *testing.T) {
	j := entities.JSONMap{"key": "value", "num": 42}
	v, err := j.Value()
	require.NoError(t, err)
	assert.Contains(t, v.(string), `"key":"value"`)
	assert.Contains(t, v.(string), `"num":42`)
}

func TestJSONMap_Value_MarshalError(t *testing.T) {
	// Channels cannot be marshalled to JSON — triggers json.Marshal failure.
	j := entities.JSONMap{"bad": make(chan int)}
	_, err := j.Value()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JSONMap.Value")
}

func TestJSONMap_Scan_Nil(t *testing.T) {
	var j entities.JSONMap
	require.NoError(t, j.Scan(nil))
	assert.NotNil(t, j)
	assert.Empty(t, j)
}

func TestJSONMap_Scan_Bytes(t *testing.T) {
	var j entities.JSONMap
	require.NoError(t, j.Scan([]byte(`{"a":"b","n":1}`)))
	assert.Equal(t, "b", j["a"])
	assert.Equal(t, float64(1), j["n"])
}

func TestJSONMap_Scan_String(t *testing.T) {
	var j entities.JSONMap
	require.NoError(t, j.Scan(`{"x":"y"}`))
	assert.Equal(t, "y", j["x"])
}

func TestJSONMap_Scan_UnsupportedType(t *testing.T) {
	var j entities.JSONMap
	err := j.Scan(12345) // int is not supported
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type")
}

func TestJSONMap_Scan_InvalidJSON(t *testing.T) {
	var j entities.JSONMap
	err := j.Scan([]byte(`not json`))
	require.Error(t, err)
}

// ─── Round-trip test ───────────────────────────────────────────────────────────

func TestJSONMap_RoundTrip(t *testing.T) {
	original := entities.JSONMap{"name": "test", "count": float64(5), "ok": true}
	v, err := original.Value()
	require.NoError(t, err)

	var restored entities.JSONMap
	require.NoError(t, restored.Scan(v))
	assert.Equal(t, original["name"], restored["name"])
	assert.Equal(t, original["count"], restored["count"])
	assert.Equal(t, original["ok"], restored["ok"])
}

// Ensure driver.Value is returned as expected.
var _ driver.Valuer = entities.JSONMap{}

// ─── Smoke test: entity construction ───────────────────────────────────────────

func TestEntities_BasicConstruction(t *testing.T) {
	// Sanity: entities can be constructed with UUIDs and basic fields.
	id := uuid.New()
	orgID := uuid.New()

	u := &entities.User{ID: id, Email: "a@b.com", Role: constants.UserRoleAdmin}
	assert.Equal(t, id, u.ID)
	assert.True(t, u.IsAdmin())

	org := &entities.Organization{ID: orgID, Name: "Test", OwnerID: id}
	assert.Equal(t, orgID, org.ID)
	assert.False(t, org.IsDeleted())
}
