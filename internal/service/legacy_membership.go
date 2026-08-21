package service

import (
	"context"
	"errors"

	"designreview/internal/domain"
)

func (service *MembershipService) addLegacyMember(ctx context.Context, projectID domain.ID, actor domain.Actor, role domain.Role) (domain.Member, error) {
	if existing, err := service.repository.FindMemberByEmail(ctx, projectID, actor.Email); err == nil {
		return domain.Member{}, domain.Wrap("add", "member", existing.ActorID.String(), domain.ErrAlreadyExists)
	} else if !errors.Is(err, domain.ErrNotFound) {
		return domain.Member{}, domain.Wrap("check uniqueness", "member", actor.ID.String(), err)
	}
	member, err := domain.NewMember(projectID, actor, role, service.clock.Now())
	if err != nil {
		return domain.Member{}, err
	}
	if err := service.repository.PutMember(ctx, member); err != nil {
		return domain.Member{}, domain.Wrap("add", "member", actor.ID.String(), err)
	}
	return member, nil
}

func equivalentLegacyEmail(left, right string) bool {
	return domain.NormalizeEmail(left) == domain.NormalizeEmail(right)
}
