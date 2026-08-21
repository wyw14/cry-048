package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type AuditEvent struct {
	ID         ID
	ProjectID  ID
	ActorID    ID
	Action     string
	EntityType string
	EntityID   ID
	OccurredAt time.Time
	Metadata   map[string]string
}

func (event AuditEvent) Clone() AuditEvent {
	clone := event
	clone.Metadata = make(map[string]string, len(event.Metadata))
	for key, value := range event.Metadata {
		clone.Metadata[key] = value
	}
	return clone
}

type Notification struct {
	ID             ID
	ProjectID      ID
	EventID        ID
	RecipientID    ID
	Template       string
	Payload        map[string]string
	Attempt        int
	LeaseOwner     string
	LeaseExpiresAt time.Time
	SentAt         *time.Time
}

func NotificationKey(eventID, recipientID ID) string {
	return eventID.String() + ":" + recipientID.String()
}

func (notification Notification) Clone() Notification {
	clone := notification
	clone.Payload = make(map[string]string, len(notification.Payload))
	for key, value := range notification.Payload {
		clone.Payload[key] = value
	}
	return clone
}

func UniqueRecipients(values []ID) ([]ID, error) {
	return legacyRecipientPlan(values)
}

func uniqueRecipientCopy(values []ID) ([]ID, error) {
	seen := make(map[ID]struct{}, len(values))
	result := make([]ID, 0, len(values))
	for _, value := range values {
		if !value.Valid() {
			return nil, fmt.Errorf("%w: invalid recipient", ErrInvalidArgument)
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(left, right int) bool { return strings.Compare(result[left].String(), result[right].String()) < 0 })
	return result, nil
}
