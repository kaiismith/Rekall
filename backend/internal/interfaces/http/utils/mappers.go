package httputils

import (
	"github.com/rekall/backend/internal/domain/entities"
	"github.com/rekall/backend/internal/interfaces/http/dto"
)

func ToUserResponse(u *entities.User) dto.UserResponse {
	return dto.UserResponse{
		ID:            u.ID.String(),
		Email:         u.Email,
		FullName:      u.FullName,
		Role:          u.Role,
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt,
	}
}

func ToOrgResponse(o *entities.Organization) dto.OrgResponse {
	return dto.OrgResponse{
		ID:        o.ID.String(),
		Name:      o.Name,
		Slug:      o.Slug,
		OwnerID:   o.OwnerID.String(),
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
	}
}

func ToMemberResponse(m *entities.OrgMembership) dto.MemberResponse {
	return dto.MemberResponse{
		UserID:   m.UserID.String(),
		OrgID:    m.OrgID.String(),
		Role:     m.Role,
		JoinedAt: m.JoinedAt,
	}
}

func ToDeptResponse(d *entities.Department) dto.DeptResponse {
	return dto.DeptResponse{
		ID:          d.ID.String(),
		OrgID:       d.OrgID.String(),
		Name:        d.Name,
		Description: d.Description,
		CreatedBy:   d.CreatedBy.String(),
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

func ToDeptMemberResponse(m *entities.DepartmentMembership) dto.DeptMemberResponse {
	return dto.DeptMemberResponse{
		UserID:       m.UserID.String(),
		DepartmentID: m.DepartmentID.String(),
		Role:         m.Role,
		JoinedAt:     m.JoinedAt,
	}
}
