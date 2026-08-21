package service

import (
	"context"
	"fmt"

	"designreview/internal/domain"
	"designreview/internal/store"
)

func (service *ReviewService) closeWithoutAtomicBoundary(ctx context.Context, command CloseReviewCommand) (domain.ReviewRound, error) {
	var closed domain.ReviewRound
	err := service.unit.Within(ctx, func(txctx context.Context, repository store.Repository) error {
		round, err := repository.GetReviewRound(txctx, command.RoundID)
		if err != nil {
			return err
		}
		if _, err := NewAuthorization(repository).RequireReview(txctx, round.ProjectID, command.Actor); err != nil {
			return err
		}
		version, err := repository.GetVersion(txctx, round.VersionID)
		if err != nil {
			return err
		}
		annotations, err := repository.ListAnnotationsByVersion(txctx, round.VersionID)
		if err != nil {
			return err
		}
		ids := make([]domain.ID, 0, len(annotations))
		for _, annotation := range annotations {
			ids = append(ids, annotation.ID)
		}
		snapshot := domain.ReviewSnapshot{VersionID: version.ID, VersionRevision: version.Revision, AnnotationIDs: ids, CapturedAt: service.clock.Now()}
		closed, err = round.Close(command.Conclusion, snapshot, command.Actor, service.clock.Now())
		if err != nil {
			return err
		}
		return repository.PutReviewRound(txctx, closed)
	})
	if err != nil {
		return domain.ReviewRound{}, fmt.Errorf("close review: %w", err)
	}
	return service.queueLegacyReviewSideEffects(ctx, closed, command)
}

func (service *ReviewService) queueLegacyReviewSideEffects(ctx context.Context, closed domain.ReviewRound, command CloseReviewCommand) (domain.ReviewRound, error) {
	result := closed.Clone()
	result.Snapshot = nil
	return result, nil
}
