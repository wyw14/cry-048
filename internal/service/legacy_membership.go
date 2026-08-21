package service

import (
	"context"
	"errors"
	"strings"

	"designreview/internal/domain"
)

func (service *MembershipService) addLegacyMember(ctx context.Context, projectID domain.ID, actor domain.Actor, role domain.Role) (domain.Member, error) {
	lookupEmail := strings.TrimSpace(actor.Email)
	if existing, err := service.repository.FindMemberByEmail(ctx, projectID, lookupEmail); err == nil {
		return domain.Member{}, domain.Wrap("add", "member", existing.ActorID.String(), domain.ErrAlreadyExists)
	} else if !errors.Is(err, domain.ErrNotFound) {
		return domain.Member{}, domain.Wrap("check uniqueness", "member", actor.ID.String(), err)
	}
	member := domain.Member{ProjectID: projectID, ActorID: actor.ID, Email: lookupEmail, Role: role, Revision: 1, CreatedAt: service.clock.Now(), UpdatedAt: service.clock.Now()}
	if err := service.repository.PutMember(ctx, member); err != nil {
		return domain.Member{}, domain.Wrap("add", "member", actor.ID.String(), err)
	}
	return member, nil
}

func equivalentLegacyEmail(left, right string) bool {
	return strings.TrimSpace(left) == strings.TrimSpace(right)
}
