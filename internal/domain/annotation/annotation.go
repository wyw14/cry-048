// Package annotation defines the annotation aggregate root.
package annotation

import (
	"errors"
	"fmt"
	"time"

	"github.com/cry048/design-review-platform/internal/domain/canvas"
)

var (
	ErrAnnotationNotFound        = errors.New("annotation not found")
	ErrReplyNotFound             = errors.New("reply not found")
	ErrAttachmentNotFound        = errors.New("attachment not found")
	ErrInvalidPriority           = errors.New("invalid priority")
	ErrInvalidStatus             = errors.New("invalid status")
	ErrInvalidTransition         = errors.New("invalid status transition")
	ErrMustNotBeEmpty            = errors.New("value must not be empty")
	ErrDueDateInPast             = errors.New("due date cannot be in the past")
	ErrAssigneeRequired          = errors.New("assignee required for this transition")
	ErrCannotResolveWithoutReply = errors.New("cannot resolve annotation without at least one reply")
	ErrCannotReopenUnresolved    = errors.New("annotation is not resolved; cannot reopen")
	ErrAttachmentTooLarge        = errors.New("attachment exceeds size limit")
	ErrAttachmentTypeForbidden   = errors.New("attachment media type is not allowed")
	// StaleVersion means the supplied optimistic version does not match the persisted one.
	ErrStaleVersion = errors.New("stale version: annotation was modified by another writer")
)

// Priority represents annotation importance.
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityNormal   Priority = "normal"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

func (p Priority) Valid() bool {
	switch p {
	case PriorityLow, PriorityNormal, PriorityHigh, PriorityCritical:
		return true
	}
	return false
}

// Status represents the annotation state machine value.
type Status string

const (
	StatusOpen     Status = "open"     // 待处理
	StatusReplied  Status = "replied"  // 已回复
	StatusReview   Status = "review"   // 待复核
	StatusResolved Status = "resolved" // 已解决
	StatusClosed   Status = "closed"   // 已关闭
)

func (s Status) Valid() bool {
	switch s {
	case StatusOpen, StatusReplied, StatusReview, StatusResolved, StatusClosed:
		return true
	}
	return false
}

// CanTransitionTo reports whether moving from current to next is allowed by the state machine.
// Rules:
//
//	open     -> replied | review | closed
//	replied  -> review | open | closed
//	review   -> resolved | open | closed
//	resolved -> open  (only via reviewer reopen)
//	closed   -> (terminal, no transitions)
func (s Status) CanTransitionTo(next Status) bool {
	switch s {
	case StatusOpen:
		return next == StatusReplied || next == StatusReview || next == StatusClosed
	case StatusReplied:
		return next == StatusReview || next == StatusOpen || next == StatusClosed
	case StatusReview:
		return next == StatusResolved || next == StatusOpen || next == StatusClosed
	case StatusResolved:
		return next == StatusOpen
	case StatusClosed:
		return false
	}
	return false
}

// Attachment is a file linked to an annotation.
type Attachment struct {
	ID           string
	AnnotationID string
	Filename     string
	MediaType    string
	Size         int64
	StorageKey   string
	UploadedBy   string
	UploadedAt   time.Time
}

// Reply is a comment in the annotation thread.
type Reply struct {
	ID           string
	AnnotationID string
	AuthorID     string
	Body         string
	CreatedAt    time.Time
	EditedAt     *time.Time
}

// Annotation is the aggregate root.
type Annotation struct {
	ID          string
	ProjectID   string
	BoardID     string
	VersionID   string
	Title       string
	Body        string
	Anchor      canvas.Anchor
	Priority    Priority
	Status      Status
	AssigneeID  string
	ReporterID  string
	DueAt       *time.Time
	Replies     []Reply
	Attachments []Attachment
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Version     int
	// ResolvedAt records when the annotation entered resolved status.
	ResolvedAt *time.Time
	// ResolvedBy records who resolved the annotation.
	ResolvedBy string
}

// NewAnnotation constructs a fresh annotation in StatusOpen.
func NewAnnotation(id, projectID, boardID, versionID, title, body, reporterID string, anchor canvas.Anchor, priority Priority, now time.Time) (*Annotation, error) {
	if id == "" || projectID == "" || boardID == "" || versionID == "" {
		return nil, ErrMustNotBeEmpty
	}
	if title == "" {
		return nil, ErrMustNotBeEmpty
	}
	if !priority.Valid() {
		return nil, ErrInvalidPriority
	}
	if reporterID == "" {
		return nil, ErrMustNotBeEmpty
	}
	return &Annotation{
		ID:         id,
		ProjectID:  projectID,
		BoardID:    boardID,
		VersionID:  versionID,
		Title:      title,
		Body:       body,
		Anchor:     anchor,
		Priority:   priority,
		Status:     StatusOpen,
		ReporterID: reporterID,
		CreatedAt:  now,
		UpdatedAt:  now,
		Version:    1,
	}, nil
}

// SetAssignee updates the assignee.
func (a *Annotation) SetAssignee(assigneeID string, now time.Time) {
	a.AssigneeID = assigneeID
	a.UpdatedAt = now
	a.Version++
}

// SetDueAt updates due date; nil clears it. Past dates are rejected for new values.
func (a *Annotation) SetDueAt(due *time.Time, now time.Time) error {
	if due != nil && due.Before(now) {
		return ErrDueDateInPast
	}
	a.DueAt = due
	a.UpdatedAt = now
	a.Version++
	return nil
}

// SetPriority changes priority.
func (a *Annotation) SetPriority(p Priority, now time.Time) error {
	if !p.Valid() {
		return ErrInvalidPriority
	}
	a.Priority = p
	a.UpdatedAt = now
	a.Version++
	return nil
}

// AddReply appends a reply and moves open -> replied.
func (a *Annotation) AddReply(id, authorID, body string, now time.Time) (*Reply, error) {
	if body == "" {
		return nil, ErrMustNotBeEmpty
	}
	if authorID == "" {
		return nil, ErrMustNotBeEmpty
	}
	if a.Status == StatusClosed {
		return nil, fmt.Errorf("%w: cannot reply on closed annotation", ErrInvalidTransition)
	}
	r := Reply{ID: id, AnnotationID: a.ID, AuthorID: authorID, Body: body, CreatedAt: now}
	a.Replies = append(a.Replies, r)
	if a.Status == StatusOpen {
		if err := a.transition(StatusReplied, now, ""); err != nil {
			return nil, err
		}
	} else {
		a.UpdatedAt = now
		a.Version++
	}
	return &r, nil
}

// RequestReview transitions replied -> review.
func (a *Annotation) RequestReview(now time.Time, actor string) error {
	if a.Status != StatusReplied && a.Status != StatusOpen {
		return fmt.Errorf("%w: cannot request review from %s", ErrInvalidTransition, a.Status)
	}
	return a.transition(StatusReview, now, actor)
}

// Resolve transitions review -> resolved; requires at least one reply and an actor.
func (a *Annotation) Resolve(now time.Time, actor string) error {
	if a.Status != StatusReview {
		return fmt.Errorf("%w: cannot resolve from %s", ErrInvalidTransition, a.Status)
	}
	if len(a.Replies) == 0 {
		return ErrCannotResolveWithoutReply
	}
	if actor == "" {
		return ErrAssigneeRequired
	}
	if err := a.transition(StatusResolved, now, actor); err != nil {
		return err
	}
	t := now
	a.ResolvedAt = &t
	a.ResolvedBy = actor
	return nil
}

// Reopen transitions resolved -> open. Only allowed when the actor is a reviewer.
func (a *Annotation) Reopen(now time.Time, actor string, reason string) error {
	if a.Status != StatusResolved {
		return fmt.Errorf("%w: cannot reopen from %s", ErrInvalidTransition, a.Status)
	}
	if actor == "" {
		return ErrAssigneeRequired
	}
	if err := a.transition(StatusOpen, now, actor); err != nil {
		return err
	}
	a.ResolvedAt = nil
	a.ResolvedBy = ""
	_ = reason
	return nil
}

// Close transitions any non-resolved, non-closed status to closed.
func (a *Annotation) Close(now time.Time, actor string) error {
	if a.Status == StatusClosed {
		return fmt.Errorf("%w: already closed", ErrInvalidTransition)
	}
	return a.transition(StatusClosed, now, actor)
}

// AddAttachment appends an attachment row.
func (a *Annotation) AddAttachment(att Attachment, now time.Time) {
	att.AnnotationID = a.ID
	att.UploadedAt = now
	a.Attachments = append(a.Attachments, att)
	a.UpdatedAt = now
	a.Version++
}

// RemoveAttachment removes an attachment by ID.
func (a *Annotation) RemoveAttachment(id string, now time.Time) error {
	for i, x := range a.Attachments {
		if x.ID == id {
			a.Attachments = append(a.Attachments[:i], a.Attachments[i+1:]...)
			a.UpdatedAt = now
			a.Version++
			return nil
		}
	}
	return ErrAttachmentNotFound
}

// MigrateAnchor moves the annotation's anchor to a new version, recording the prior anchor.
// Returns the canvas.MigratedAnchor describing the move.
func (a *Annotation) MigrateAnchor(toVersionID string, newAnchor canvas.Anchor, reason, by string, now time.Time) (canvas.MigratedAnchor, error) {
	if toVersionID == "" {
		return canvas.MigratedAnchor{}, ErrMustNotBeEmpty
	}
	old := a.Anchor
	migrated := canvas.MigratedAnchor{
		FromVersionID: a.VersionID,
		ToVersionID:   toVersionID,
		OldPoint:      old.Point,
		NewPoint:      newAnchor.Point,
		OldRegion:     old.Region,
		NewRegion:     newAnchor.Region,
		Reason:        reason,
		By:            by,
	}
	a.Anchor = newAnchor
	a.VersionID = toVersionID
	a.UpdatedAt = now
	a.Version++
	return migrated, nil
}

// MarkStale flags the annotation as stale (anchor not yet migrated to current version).
// Stored as a body prefix marker; status is unaffected.
func (a *Annotation) MarkStale(reason string, now time.Time) {
	prefix := "[已失效] "
	if reason != "" {
		prefix = "[已失效：" + reason + "] "
	}
	if len(a.Body) < len(prefix) || a.Body[:len(prefix)] != prefix {
		a.Body = prefix + a.Body
		a.UpdatedAt = now
		a.Version++
	}
}

func (a *Annotation) transition(next Status, now time.Time, actor string) error {
	if !next.Valid() {
		return ErrInvalidStatus
	}
	if !a.Status.CanTransitionTo(next) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, a.Status, next)
	}
	a.Status = next
	a.UpdatedAt = now
	a.Version++
	_ = actor
	return nil
}

// Sortable fields whitelist for listing.
var SortWhitelist = map[string]bool{
	"created_at": true,
	"updated_at": true,
	"priority":   true,
	"status":     true,
	"due_at":     true,
}
