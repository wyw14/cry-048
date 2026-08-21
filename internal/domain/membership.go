package domain

import (
	"fmt"
	"strings"
	"time"
)

type Role string

const (
	RoleViewer   Role = "viewer"
	RoleReviewer Role = "reviewer"
	RoleEditor   Role = "editor"
	RoleOwner    Role = "owner"
)

func (role Role) CanEdit() bool {
	return role == RoleEditor || role == RoleOwner
}

func (role Role) CanReview() bool {
	return role == RoleReviewer || role == RoleEditor || role == RoleOwner
}

func (role Role) Valid() bool {
	return role == RoleViewer || role == RoleReviewer || role == RoleEditor || role == RoleOwner
}

type Member struct {
	ProjectID ID
	ActorID   ID
	Email     string
	Role      Role
	Revision  Revision
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewMember(projectID ID, actor Actor, role Role, now time.Time) (Member, error) {
	email := strings.ToLower(strings.TrimSpace(actor.Email))
	if !projectID.Valid() || !actor.Valid() || !role.Valid() {
		return Member{}, fmt.Errorf("%w: invalid member", ErrInvalidArgument)
	}
	return Member{
		ProjectID: projectID, ActorID: actor.ID, Email: email, Role: role,
		Revision: 1, CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (member Member) ChangeRole(role Role, expected Revision, now time.Time) (Member, error) {
	if !member.Revision.Matches(expected) {
		return Member{}, ErrConflict
	}
	if !role.Valid() || member.Role == RoleOwner {
		return Member{}, fmt.Errorf("%w: role cannot be changed", ErrForbidden)
	}
	result := member
	result.Role = role
	result.Revision = member.Revision.Next()
	result.UpdatedAt = now
	return result, nil
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
