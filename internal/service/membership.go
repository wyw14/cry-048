package service

import (
	"context"
	"errors"

	"designreview/internal/domain"
	"designreview/internal/store"
)

type MembershipService struct {
	repository store.Repository
	clock      domain.Clock
}

func NewMembershipService(repository store.Repository, clock domain.Clock) *MembershipService {
	return &MembershipService{repository: repository, clock: clock}
}

func (service *MembershipService) Add(ctx context.Context, projectID domain.ID, actor domain.Actor, role domain.Role) (domain.Member, error) {
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

func (service *MembershipService) ChangeRole(ctx context.Context, projectID, actorID domain.ID, role domain.Role, expected domain.Revision) (domain.Member, error) {
	member, err := service.repository.GetMember(ctx, projectID, actorID)
	if err != nil {
		return domain.Member{}, domain.Wrap("load", "member", actorID.String(), err)
	}
	updated, err := member.ChangeRole(role, expected, service.clock.Now())
	if err != nil {
		return domain.Member{}, domain.Wrap("change role", "member", actorID.String(), err)
	}
	if err := service.repository.CompareAndSwapMember(ctx, updated, expected); err != nil {
		return domain.Member{}, domain.Wrap("commit role", "member", actorID.String(), err)
	}
	return updated, nil
}
