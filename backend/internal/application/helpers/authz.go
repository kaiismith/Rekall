package helpers

import (
	"github.com/rekall/backend/internal/domain/entities"
)

// IsPlatformAdmin reports whether the user holds the platform-level admin role.
// Nil-safe — a missing user is never an admin.
func IsPlatformAdmin(u *entities.User) bool {
	return u != nil && u.IsAdmin()
}

// CanManageOrg reports whether the caller can rename/delete the organization
// or its departments. Platform admins always can; otherwise the caller must
// be an org owner or admin (CanManageOrg() on the membership).
func CanManageOrg(caller *entities.User, orgMem *entities.OrgMembership) bool {
	if IsPlatformAdmin(caller) {
		return true
	}
	return orgMem != nil && orgMem.CanManageOrg()
}

// CanManageDepartment is the metadata-management predicate (rename, delete,
// create-sibling). Identical to CanManageOrg today; named separately so the
// caller's intent is legible and future divergence is easy.
func CanManageDepartment(caller *entities.User, orgMem *entities.OrgMembership) bool {
	return CanManageOrg(caller, orgMem)
}

// CanManageDepartmentMembers reports whether the caller can add, remove, or
// change roles of members in the given department. Platform admins, org
// owners/admins, and dept heads all qualify.
func CanManageDepartmentMembers(
	caller *entities.User,
	orgMem *entities.OrgMembership,
	deptMem *entities.DepartmentMembership,
) bool {
	if IsPlatformAdmin(caller) {
		return true
	}
	if orgMem != nil && orgMem.CanManageOrg() {
		return true
	}
	return deptMem != nil && deptMem.CanManageMembers()
}
