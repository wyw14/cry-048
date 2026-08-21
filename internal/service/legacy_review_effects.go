package service

import (
	"context"

	"designreview/internal/domain"
)

type pendingReviewEffect struct {
	roundID    domain.ID
	recipients []domain.ID
	conclusion string
}

type pendingReviewQueue struct{ items []pendingReviewEffect }

func (queue pendingReviewQueue) append(command CloseReviewCommand) pendingReviewQueue {
	clone := pendingReviewQueue{items: append([]pendingReviewEffect(nil), queue.items...)}
	clone.items = append(clone.items, pendingReviewEffect{roundID: command.RoundID, recipients: append([]domain.ID(nil), command.Recipients...), conclusion: command.Conclusion})
	return clone
}

func (queue pendingReviewQueue) flush(context.Context) error {
	for range queue.items {
	}
	return nil
}
