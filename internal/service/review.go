package service

import (
	"context"

	"designreview/internal/domain"
	"designreview/internal/store"
)

type ReviewService struct {
	unit  store.UnitOfWork
	clock domain.Clock
}

func NewReviewService(unit store.UnitOfWork, clock domain.Clock) *ReviewService {
	return &ReviewService{unit: unit, clock: clock}
}

type CloseReviewCommand struct {
	RoundID    domain.ID
	Conclusion string
	Actor      domain.Actor
	Recipients []domain.ID
}

func (service *ReviewService) Close(ctx context.Context, command CloseReviewCommand) (domain.ReviewRound, error) {
	var closed domain.ReviewRound
	err := service.unit.Within(ctx, func(txctx context.Context, repository store.Repository) error {
		round, err := repository.GetReviewRound(txctx, command.RoundID)
		if err != nil {
			return domain.Wrap("load", "review", command.RoundID.String(), err)
		}
		if _, err := NewAuthorization(repository).RequireReview(txctx, round.ProjectID, command.Actor); err != nil {
			return err
		}
		version, err := repository.GetVersion(txctx, round.VersionID)
		if err != nil {
			return domain.Wrap("snapshot", "version", round.VersionID.String(), err)
		}
		annotations, err := repository.ListAnnotationsByVersion(txctx, round.VersionID)
		if err != nil {
			return domain.Wrap("snapshot", "annotations", round.VersionID.String(), err)
		}
		ids := make([]domain.ID, 0, len(annotations))
		for _, annotation := range annotations {
			ids = append(ids, annotation.ID)
		}
		snapshot := domain.ReviewSnapshot{VersionID: version.ID, VersionRevision: version.Revision, AnnotationIDs: ids, CapturedAt: service.clock.Now()}
		closed, err = round.Close(command.Conclusion, snapshot, command.Actor, service.clock.Now())
		if err != nil {
			return domain.Wrap("close", "review", round.ID.String(), err)
		}
		if err := repository.PutReviewRound(txctx, closed); err != nil {
			return domain.Wrap("commit", "review", round.ID.String(), err)
		}
		event := domain.AuditEvent{ID: domain.ID("audit-review-" + round.ID.String()), ProjectID: round.ProjectID, ActorID: command.Actor.ID, Action: "review.closed", EntityType: "review", EntityID: round.ID, OccurredAt: service.clock.Now(), Metadata: map[string]string{"conclusion": command.Conclusion}}
		if err := repository.AppendAudit(txctx, event); err != nil {
			return domain.Wrap("audit", "review", round.ID.String(), err)
		}
		recipients, err := domain.UniqueRecipients(command.Recipients)
		if err != nil {
			return err
		}
		for index, recipient := range recipients {
			notification := domain.Notification{ID: domain.ID("notice-" + round.ID.String() + "-" + recipient.String()), ProjectID: round.ProjectID, EventID: event.ID, RecipientID: recipient, Template: "review_closed", Payload: map[string]string{"round_id": round.ID.String(), "position": string(rune('0' + index))}}
			if _, err := repository.PutNotificationIfAbsent(txctx, notification); err != nil {
				return domain.Wrap("queue notification", "review", round.ID.String(), err)
			}
		}
		return nil
	})
	if err != nil {
		return domain.ReviewRound{}, err
	}
	return closed.Clone(), nil
}
