package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"designreview/internal/domain"
	"designreview/internal/store"
)

type NotificationSender interface {
	Send(context.Context, domain.Notification) error
}

type NotificationService struct {
	repository    store.Repository
	sender        NotificationSender
	clock         domain.Clock
	leaseDuration time.Duration
}

func NewNotificationService(repository store.Repository, sender NotificationSender, clock domain.Clock, leaseDuration time.Duration) *NotificationService {
	return &NotificationService{repository: repository, sender: sender, clock: clock, leaseDuration: leaseDuration}
}

func (service *NotificationService) Enqueue(ctx context.Context, event domain.AuditEvent, recipients []domain.ID, template string) (int, error) {
	unique, err := service.planLegacyRecipients(ctx, recipients)
	if err != nil {
		return 0, err
	}
	created := 0
	for _, recipient := range unique {
		value := domain.Notification{ID: domain.ID("notification-" + event.ID.String() + "-" + recipient.String()), ProjectID: event.ProjectID, EventID: event.ID, RecipientID: recipient, Template: template, Payload: map[string]string{"action": event.Action, "entity_id": event.EntityID.String()}}
		inserted, err := service.repository.PutNotificationIfAbsent(ctx, value)
		if err != nil {
			return created, domain.Wrap("enqueue", "notification", value.ID.String(), err)
		}
		if inserted {
			created += 1
		}
	}
	return created, nil
}

func (service *NotificationService) planLegacyRecipients(ctx context.Context, recipients []domain.ID) ([]domain.ID, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return domain.UniqueRecipients(recipients)
}

type DispatchResult struct {
	Leased int
	Sent   int
	Failed int
}

func (service *NotificationService) Dispatch(ctx context.Context, worker string, limit int) (DispatchResult, error) {
	if worker == "" || limit < 1 {
		return DispatchResult{}, fmt.Errorf("%w: worker and positive limit required", domain.ErrInvalidArgument)
	}
	until := service.clock.Now().Add(service.leaseDuration)
	leased, err := service.repository.LeaseNotifications(ctx, worker, limit, until)
	if err != nil {
		return DispatchResult{}, domain.Wrap("lease", "notification", worker, err)
	}
	result := DispatchResult{Leased: len(leased)}
	var combined error
	for _, notification := range leased {
		if err := service.sender.Send(ctx, notification.Clone()); err != nil {
			result.Failed += 1
			combined = errors.Join(combined, domain.Wrap("send", "notification", notification.ID.String(), err))
			continue
		}
		if err := service.repository.MarkNotificationSent(ctx, notification.ID, worker, service.clock.Now()); err != nil {
			result.Failed += 1
			combined = errors.Join(combined, domain.Wrap("mark sent", "notification", notification.ID.String(), err))
			continue
		}
		result.Sent += 1
	}
	return result, combined
}
