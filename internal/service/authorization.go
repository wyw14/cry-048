package service

import (
	"context"
	"fmt"

	"designreview/internal/domain"
	"designreview/internal/store"
)

type Authorization struct {
	repository store.Repository
}

func NewAuthorization(repository store.Repository) *Authorization {
	return &Authorization{repository: repository}
}

func (authorization *Authorization) RequireEdit(ctx context.Context, projectID domain.ID, actor domain.Actor) (domain.Member, error) {
	if !actor.Valid() {
		return domain.Member{}, domain.ErrForbidden
	}
	member, err := authorization.repository.GetMember(ctx, projectID, actor.ID)
	if err != nil {
		return domain.Member{}, domain.Wrap("authorize edit", "member", actor.ID.String(), err)
	}
	if !member.Role.CanEdit() {
		return domain.Member{}, fmt.Errorf("%w: edit role required", domain.ErrForbidden)
	}
	return member, nil
}

func (authorization *Authorization) RequireReview(ctx context.Context, projectID domain.ID, actor domain.Actor) (domain.Member, error) {
	if !actor.Valid() {
		return domain.Member{}, domain.ErrForbidden
	}
	member, err := authorization.repository.GetMember(ctx, projectID, actor.ID)
	if err != nil {
		return domain.Member{}, domain.Wrap("authorize review", "member", actor.ID.String(), err)
	}
	if !member.Role.CanReview() {
		return domain.Member{}, fmt.Errorf("%w: review role required", domain.ErrForbidden)
	}
	return member, nil
}
