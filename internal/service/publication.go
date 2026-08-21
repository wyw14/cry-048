package service

import (
	"context"
	"fmt"

	"designreview/internal/domain"
	"designreview/internal/store"
)

type PublicationService struct {
	unit  store.UnitOfWork
	clock domain.Clock
}

func NewPublicationService(unit store.UnitOfWork, clock domain.Clock) *PublicationService {
	return &PublicationService{unit: unit, clock: clock}
}

func (service *PublicationService) Publish(ctx context.Context, versionID domain.ID, actor domain.Actor) (domain.DesignVersion, error) {
	var published domain.DesignVersion
	err := service.unit.Within(ctx, func(txctx context.Context, repository store.Repository) error {
		version, err := repository.GetVersion(txctx, versionID)
		if err != nil {
			return domain.Wrap("load", "version", versionID.String(), err)
		}
		authorization := NewAuthorization(repository)
		board, err := repository.GetBoard(txctx, version.BoardID)
		if err != nil {
			return domain.Wrap("load", "board", version.BoardID.String(), err)
		}
		if _, err := authorization.RequireEdit(txctx, board.ProjectID, actor); err != nil {
			return err
		}
		versions, err := repository.ListVersionsByBoard(txctx, version.BoardID)
		if err != nil {
			return domain.Wrap("list", "version", version.BoardID.String(), err)
		}
		for _, candidate := range versions {
			if candidate.Status != domain.VersionPublished {
				continue
			}
			updated, err := candidate.Supersede(version.ID)
			if err != nil {
				return domain.Wrap("supersede", "version", candidate.ID.String(), err)
			}
			if err := repository.PutVersion(txctx, updated); err != nil {
				return domain.Wrap("commit supersede", "version", candidate.ID.String(), err)
			}
		}
		published, err = version.Publish(service.clock.Now())
		if err != nil {
			return domain.Wrap("publish", "version", version.ID.String(), err)
		}
		if err := repository.PutVersion(txctx, published); err != nil {
			return domain.Wrap("commit publish", "version", version.ID.String(), err)
		}
		event := domain.AuditEvent{ID: domain.ID("audit-publish-" + version.ID.String()), ProjectID: board.ProjectID, ActorID: actor.ID, Action: "version.published", EntityType: "version", EntityID: version.ID, OccurredAt: service.clock.Now(), Metadata: map[string]string{"board_id": version.BoardID.String()}}
		if err := repository.AppendAudit(txctx, event); err != nil {
			return domain.Wrap("audit publish", "version", version.ID.String(), err)
		}
		return nil
	})
	if err != nil {
		return domain.DesignVersion{}, fmt.Errorf("publication failed: %w", err)
	}
	return published, nil
}
