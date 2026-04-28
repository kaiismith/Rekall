package helpers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rekall/backend/internal/application/helpers"
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/pkg/constants"
)

func adminUser() *entities.User  { return &entities.User{Role: "admin"} }
func memberUser() *entities.User { return &entities.User{Role: "member"} }

func TestIsPlatformAdmin(t *testing.T) {
	assert.True(t, helpers.IsPlatformAdmin(adminUser()))
	assert.False(t, helpers.IsPlatformAdmin(memberUser()))
	assert.False(t, helpers.IsPlatformAdmin(nil))
}

func TestCanManageOrg_PlatformAdminFallthrough(t *testing.T) {
	// Platform admin without org membership still passes.
	assert.True(t, helpers.CanManageOrg(adminUser(), nil))
}

func TestCanManageOrg_OrgOwnerOrAdmin(t *testing.T) {
	owner := &entities.OrgMembership{Role: constants.OrgRoleOwner}
	admin := &entities.OrgMembership{Role: constants.OrgRoleAdmin}
	member := &entities.OrgMembership{Role: constants.OrgRoleMember}

	assert.True(t, helpers.CanManageOrg(memberUser(), owner))
	assert.True(t, helpers.CanManageOrg(memberUser(), admin))
	assert.False(t, helpers.CanManageOrg(memberUser(), member))
	assert.False(t, helpers.CanManageOrg(memberUser(), nil))
}

func TestCanManageDepartmentMembers_AllPaths(t *testing.T) {
	orgAdmin := &entities.OrgMembership{Role: constants.OrgRoleAdmin}
	orgMember := &entities.OrgMembership{Role: constants.OrgRoleMember}
	deptHead := &entities.DepartmentMembership{Role: constants.DeptRoleHead}
	deptMember := &entities.DepartmentMembership{Role: constants.DeptRoleMember}

	// Platform admin always allowed
	assert.True(t, helpers.CanManageDepartmentMembers(adminUser(), nil, nil))
	// Org admin allowed
	assert.True(t, helpers.CanManageDepartmentMembers(memberUser(), orgAdmin, deptMember))
	// Plain org member but dept head — allowed
	assert.True(t, helpers.CanManageDepartmentMembers(memberUser(), orgMember, deptHead))
	// Plain org + plain dept member — denied
	assert.False(t, helpers.CanManageDepartmentMembers(memberUser(), orgMember, deptMember))
	// No memberships at all — denied
	assert.False(t, helpers.CanManageDepartmentMembers(memberUser(), nil, nil))
}
