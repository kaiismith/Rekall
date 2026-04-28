package repositories_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/infrastructure/repositories"
	apperr "github.com/rekall/backend/pkg/errors"
)

// ─── OrganizationRepository ──────────────────────────────────────────────────

func TestNewOrganizationRepository(t *testing.T) {
	db, _ := newMockDB(t)
	assert.NotNil(t, repositories.NewOrganizationRepository(db))
}

func TestOrgRepo_Create_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewOrganizationRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "organizations"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	mock.ExpectCommit()

	org := &entities.Organization{Name: "Acme", Slug: "acme", OwnerID: uuid.New()}
	result, err := repo.Create(context.Background(), org)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestOrgRepo_Create_Error(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewOrganizationRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "organizations"`)).WillReturnError(assert.AnError)
	mock.ExpectRollback()

	_, err := repo.Create(context.Background(), &entities.Organization{Name: "X"})
	require.Error(t, err)
}

func TestOrgRepo_GetByID_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewOrganizationRepository(db)

	id := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "organizations" WHERE id = $1 AND deleted_at IS NULL`)).
		WithArgs(id, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "slug", "owner_id", "created_at", "updated_at",
		}).AddRow(id, "Acme", "acme", uuid.New(), time.Now(), time.Now()))

	org, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, id, org.ID)
}

func TestOrgRepo_GetByID_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewOrganizationRepository(db)

	id := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "organizations"`)).
		WithArgs(id, sqlmock.AnyArg()).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := repo.GetByID(context.Background(), id)
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestOrgRepo_GetBySlug_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewOrganizationRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "organizations" WHERE slug = $1 AND deleted_at IS NULL`)).
		WithArgs("acme", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "slug", "owner_id", "created_at", "updated_at",
		}).AddRow(uuid.New(), "Acme", "acme", uuid.New(), time.Now(), time.Now()))

	org, err := repo.GetBySlug(context.Background(), "acme")
	require.NoError(t, err)
	assert.Equal(t, "acme", org.Slug)
}

func TestOrgRepo_GetBySlug_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewOrganizationRepository(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "organizations"`)).
		WithArgs("missing", sqlmock.AnyArg()).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := repo.GetBySlug(context.Background(), "missing")
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestOrgRepo_Update_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewOrganizationRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "organizations"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	org := &entities.Organization{ID: uuid.New(), Name: "Updated", Slug: "updated", OwnerID: uuid.New()}
	_, err := repo.Update(context.Background(), org)
	require.NoError(t, err)
}

func TestOrgRepo_Update_Error(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewOrganizationRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "organizations"`)).WillReturnError(assert.AnError)
	mock.ExpectRollback()

	_, err := repo.Update(context.Background(), &entities.Organization{ID: uuid.New()})
	require.Error(t, err)
}

func TestOrgRepo_SoftDelete_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewOrganizationRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "organizations" SET "deleted_at"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.SoftDelete(context.Background(), uuid.New()))
}

func TestOrgRepo_ListByUserID_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewOrganizationRepository(db)

	userID := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`JOIN org_memberships`)).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "slug", "owner_id", "created_at", "updated_at",
		}).AddRow(uuid.New(), "Acme", "acme", userID, time.Now(), time.Now()))

	orgs, err := repo.ListByUserID(context.Background(), userID)
	require.NoError(t, err)
	assert.Len(t, orgs, 1)
}

// ─── OrgMembershipRepository ──────────────────────────────────────────────────

func TestNewOrgMembershipRepository(t *testing.T) {
	db, _ := newMockDB(t)
	assert.NotNil(t, repositories.NewOrgMembershipRepository(db))
}

func TestOrgMembership_Create_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewOrgMembershipRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "org_memberships"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "joined_at"}).AddRow(uuid.New(), time.Now()))
	mock.ExpectCommit()

	m := &entities.OrgMembership{OrgID: uuid.New(), UserID: uuid.New(), Role: "owner"}
	require.NoError(t, repo.Create(context.Background(), m))
}

func TestOrgMembership_GetByOrgAndUser_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewOrgMembershipRepository(db)

	orgID, userID := uuid.New(), uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "org_memberships" WHERE org_id = $1 AND user_id = $2`)).
		WithArgs(orgID, userID, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "org_id", "user_id", "role", "joined_at"}).
			AddRow(uuid.New(), orgID, userID, "admin", time.Now()))

	m, err := repo.GetByOrgAndUser(context.Background(), orgID, userID)
	require.NoError(t, err)
	assert.Equal(t, "admin", m.Role)
}

func TestOrgMembership_GetByOrgAndUser_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewOrgMembershipRepository(db)

	orgID, userID := uuid.New(), uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "org_memberships"`)).
		WithArgs(orgID, userID, sqlmock.AnyArg()).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := repo.GetByOrgAndUser(context.Background(), orgID, userID)
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestOrgMembership_ListByOrg_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewOrgMembershipRepository(db)

	orgID := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "org_memberships" WHERE org_id = $1`)).
		WithArgs(orgID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "org_id", "user_id", "role", "joined_at"}).
			AddRow(uuid.New(), orgID, uuid.New(), "owner", time.Now()))

	members, err := repo.ListByOrg(context.Background(), orgID)
	require.NoError(t, err)
	assert.Len(t, members, 1)
}

func TestOrgMembership_Update_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewOrgMembershipRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "org_memberships"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	m := &entities.OrgMembership{ID: uuid.New(), OrgID: uuid.New(), UserID: uuid.New(), Role: "member"}
	require.NoError(t, repo.Update(context.Background(), m))
}

func TestOrgMembership_Delete_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewOrgMembershipRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "org_memberships"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.Delete(context.Background(), uuid.New(), uuid.New()))
}

// ─── DepartmentRepository ─────────────────────────────────────────────────────

func TestNewDepartmentRepository(t *testing.T) {
	db, _ := newMockDB(t)
	assert.NotNil(t, repositories.NewDepartmentRepository(db))
}

func TestDept_Create_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewDepartmentRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "departments"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	mock.ExpectCommit()

	dept := &entities.Department{OrgID: uuid.New(), Name: "Eng", CreatedBy: uuid.New()}
	_, err := repo.Create(context.Background(), dept)
	require.NoError(t, err)
}

func TestDept_Create_Error(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewDepartmentRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "departments"`)).WillReturnError(assert.AnError)
	mock.ExpectRollback()

	_, err := repo.Create(context.Background(), &entities.Department{Name: "X"})
	require.Error(t, err)
}

func TestDept_GetByID_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewDepartmentRepository(db)

	id := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "departments" WHERE id = $1 AND deleted_at IS NULL`)).
		WithArgs(id, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "org_id", "name", "description", "created_by", "created_at", "updated_at",
		}).AddRow(id, uuid.New(), "Eng", "", uuid.New(), time.Now(), time.Now()))

	dept, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, id, dept.ID)
}

func TestDept_GetByID_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewDepartmentRepository(db)

	id := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "departments"`)).
		WithArgs(id, sqlmock.AnyArg()).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := repo.GetByID(context.Background(), id)
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestDept_ListByOrg_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewDepartmentRepository(db)

	orgID := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "departments" WHERE org_id = $1 AND deleted_at IS NULL`)).
		WithArgs(orgID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "org_id", "name", "description", "created_by", "created_at", "updated_at",
		}).AddRow(uuid.New(), orgID, "Eng", "", uuid.New(), time.Now(), time.Now()))

	depts, err := repo.ListByOrg(context.Background(), orgID)
	require.NoError(t, err)
	assert.Len(t, depts, 1)
}

func TestDept_Update_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewDepartmentRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "departments"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	dept := &entities.Department{ID: uuid.New(), OrgID: uuid.New(), Name: "Updated", CreatedBy: uuid.New()}
	_, err := repo.Update(context.Background(), dept)
	require.NoError(t, err)
}

func TestDept_Update_Error(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewDepartmentRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "departments"`)).WillReturnError(assert.AnError)
	mock.ExpectRollback()

	_, err := repo.Update(context.Background(), &entities.Department{ID: uuid.New(), Name: "X"})
	require.Error(t, err)
}

func TestDept_SoftDelete_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewDepartmentRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "departments" SET "deleted_at"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.SoftDelete(context.Background(), uuid.New()))
}

func TestDept_SoftDelete_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewDepartmentRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "departments" SET "deleted_at"`)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	err := repo.SoftDelete(context.Background(), uuid.New())
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestDept_SoftDelete_Error(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewDepartmentRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "departments" SET "deleted_at"`)).
		WillReturnError(assert.AnError)
	mock.ExpectRollback()

	require.Error(t, repo.SoftDelete(context.Background(), uuid.New()))
}

// ─── DepartmentMembershipRepository ───────────────────────────────────────────

func TestNewDepartmentMembershipRepository(t *testing.T) {
	db, _ := newMockDB(t)
	assert.NotNil(t, repositories.NewDepartmentMembershipRepository(db))
}

func TestDeptMember_Create_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewDepartmentMembershipRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "department_memberships"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "joined_at"}).AddRow(uuid.New(), time.Now()))
	mock.ExpectCommit()

	m := &entities.DepartmentMembership{DepartmentID: uuid.New(), UserID: uuid.New(), Role: "member"}
	require.NoError(t, repo.Create(context.Background(), m))
}

func TestDeptMember_GetByDeptAndUser_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewDepartmentMembershipRepository(db)

	deptID, userID := uuid.New(), uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "department_memberships" WHERE department_id = $1 AND user_id = $2`)).
		WithArgs(deptID, userID, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "department_id", "user_id", "role", "joined_at"}).
			AddRow(uuid.New(), deptID, userID, "head", time.Now()))

	m, err := repo.GetByDeptAndUser(context.Background(), deptID, userID)
	require.NoError(t, err)
	assert.Equal(t, "head", m.Role)
}

func TestDeptMember_GetByDeptAndUser_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewDepartmentMembershipRepository(db)

	deptID, userID := uuid.New(), uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "department_memberships"`)).
		WithArgs(deptID, userID, sqlmock.AnyArg()).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := repo.GetByDeptAndUser(context.Background(), deptID, userID)
	require.Error(t, err)
	assert.True(t, apperr.IsNotFound(err))
}

func TestDeptMember_ListByDept_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewDepartmentMembershipRepository(db)

	deptID := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "department_memberships" WHERE department_id = $1`)).
		WithArgs(deptID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "department_id", "user_id", "role", "joined_at"}).
			AddRow(uuid.New(), deptID, uuid.New(), "member", time.Now()))

	members, err := repo.ListByDept(context.Background(), deptID)
	require.NoError(t, err)
	assert.Len(t, members, 1)
}

func TestDeptMember_Update_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewDepartmentMembershipRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "department_memberships"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	m := &entities.DepartmentMembership{ID: uuid.New(), DepartmentID: uuid.New(), UserID: uuid.New(), Role: "head"}
	require.NoError(t, repo.Update(context.Background(), m))
}

func TestDeptMember_Delete_Success(t *testing.T) {
	db, mock := newMockDB(t)
	repo := repositories.NewDepartmentMembershipRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "department_memberships"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.Delete(context.Background(), uuid.New(), uuid.New()))
}
