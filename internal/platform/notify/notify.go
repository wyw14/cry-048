// Package notify provides an offline notifier adapter that writes to an in-memory inbox.
package notify

import (
	"context"
	"sync"
	"time"

	"github.com/cry048/design-review-platform/internal/domain/annotation"
)

// Message is one delivered notification.
type Message struct {
	ID         string
	Recipient  string
	Subject    string
	Body       string
	EntityType string
	EntityID   string
	CreatedAt  time.Time
	Read       bool
}

// Inbox is a per-user message queue backed by memory.
type Inbox struct {
	mu       sync.Mutex
	messages []Message
}

// Notifier is the offline notification adapter.
type Notifier struct {
	inboxes map[string]*Inbox
	mu      sync.RWMutex
}

func New() *Notifier {
	return &Notifier{inboxes: make(map[string]*Inbox)}
}

func (n *Notifier) inboxFor(userID string) *Inbox {
	n.mu.RLock()
	if b, ok := n.inboxes[userID]; ok {
		n.mu.RUnlock()
		return b
	}
	n.mu.RUnlock()
	n.mu.Lock()
	defer n.mu.Unlock()
	if b, ok := n.inboxes[userID]; ok {
		return b
	}
	b := &Inbox{}
	n.inboxes[userID] = b
	return b
}

func (n *Notifier) send(userID, subject, body, entityType, entityID string) {
	b := n.inboxFor(userID)
	b.mu.Lock()
	defer b.mu.Unlock()
	b.messages = append(b.messages, Message{
		ID:         "msg-" + time.Now().Format("20060102-150405.000000") + "-" + userID,
		Recipient:  userID,
		Subject:    subject,
		Body:       body,
		EntityType: entityType,
		EntityID:   entityID,
		CreatedAt:  time.Now(),
	})
}

func (n *Notifier) NotifyAnnotationAssigned(ctx context.Context, a *annotation.Annotation) error {
	if a.AssigneeID == "" {
		return nil
	}
	n.send(a.AssigneeID, "批注已指派给你", "批注「"+a.Title+"」已指派给你", "annotation", a.ID)
	return nil
}

func (n *Notifier) NotifyAnnotationResolved(ctx context.Context, a *annotation.Annotation) error {
	if a.ReporterID != "" {
		n.send(a.ReporterID, "批注已解决", "批注「"+a.Title+"」已进入待复核", "annotation", a.ID)
	}
	return nil
}

func (n *Notifier) NotifyAnnotationReopened(ctx context.Context, a *annotation.Annotation) error {
	if a.AssigneeID != "" {
		n.send(a.AssigneeID, "批注被复核退回", "批注「"+a.Title+"」已被复核重新打开", "annotation", a.ID)
	}
	return nil
}

func (n *Notifier) NotifyDueApproaching(ctx context.Context, a *annotation.Annotation, when time.Time) error {
	if a.AssigneeID != "" {
		n.send(a.AssigneeID, "批注即将到期", "批注「"+a.Title+"」将于 "+when.Format("2006-01-02")+" 到期", "annotation", a.ID)
	}
	return nil
}

// List returns messages for a user, optionally unread-only.
func (n *Notifier) List(userID string, unreadOnly bool) []Message {
	b := n.inboxFor(userID)
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Message, 0, len(b.messages))
	for _, m := range b.messages {
		if unreadOnly && m.Read {
			continue
		}
		out = append(out, m)
	}
	return out
}

// MarkRead marks a message as read by id.
func (n *Notifier) MarkRead(userID, msgID string) bool {
	b := n.inboxFor(userID)
	b.mu.Lock()
	defer b.mu.Unlock()
	for i := range b.messages {
		if b.messages[i].ID == msgID {
			b.messages[i].Read = true
			return true
		}
	}
	return false
}

// UnreadCount returns the unread count for a user.
func (n *Notifier) UnreadCount(userID string) int {
	b := n.inboxFor(userID)
	b.mu.Lock()
	defer b.mu.Unlock()
	count := 0
	for _, m := range b.messages {
		if !m.Read {
			count++
		}
	}
	return count
}
