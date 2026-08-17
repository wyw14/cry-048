package application

import (
	"context"
	"time"

	"github.com/cry048/design-review-platform/internal/domain/annotation"
	"github.com/cry048/design-review-platform/internal/domain/canvas"
	"github.com/cry048/design-review-platform/internal/domain/project"
)

// === Project DTOs ===

type CreateProjectInput struct {
	ID          string
	Name        string
	Description string
}

type CreateProjectOutput struct {
	Project *project.Project
}

type CreateBoardInput struct {
	ID        string
	ProjectID string
	Name      string
	Width     int
	Height    int
}

type CreateVersionInput struct {
	ID         string
	BoardID    string
	Number     int
	Label      string
	PreviewKey string
	Notes      string
	CreatedBy  string
}

type AddMemberInput struct {
	ID        string
	ProjectID string
	UserID    string
	Role      string
}

// === Annotation DTOs ===

type CreateAnnotationInput struct {
	ID         string
	ProjectID  string
	BoardID    string
	VersionID  string
	Title      string
	Body       string
	ReporterID string
	Point      *canvas.Coordinate
	Region     *canvas.Region
	Priority   string
	AssigneeID string
	DueAt      *time.Time
}

type AddReplyInput struct {
	AnnotationID string
	AuthorID     string
	Body         string
}

type ResolveAnnotationInput struct {
	AnnotationID string
	ActorID      string
}

type ReopenAnnotationInput struct {
	AnnotationID string
	ActorID      string
	Reason       string
}

type CloseAnnotationInput struct {
	AnnotationID string
	ActorID      string
}

type RequestReviewInput struct {
	AnnotationID string
	ActorID      string
}

type AddAttachmentInput struct {
	AnnotationID string
	Filename     string
	MediaType    string
	Size         int64
	StorageKey   string
	UploadedBy   string
}

type MigrateAnchorsInput struct {
	FromVersionID string
	ToVersionID   string
	By            string
	Reason        string
}

// === Review DTOs ===

type CreateRoundInput struct {
	ID        string
	ProjectID string
	BoardID   string
	Title     string
}

type SnapshotInput struct {
	RoundID   string
	BoardID   string
	VersionID string
	ProjectID string
	By        string
}

type SetConclusionInput struct {
	RoundID        string
	Conclusion     string
	Recommendation string
	ActorID        string
}

type CloseRoundInput struct {
	RoundID string
	ActorID string
}

// === Filter for audit ===

type AuditFilter struct {
	ActorID    string
	EntityType string
	EntityID   string
	From       time.Time
	To         time.Time
}

// AttachmentValidator checks size and media type constraints.
type AttachmentValidator interface {
	ValidateUpload(mediaType string, size int64) error
}

// Notifications are sent on key transitions.
type Notifier interface {
	NotifyAnnotationAssigned(ctx context.Context, a *annotation.Annotation) error
	NotifyAnnotationResolved(ctx context.Context, a *annotation.Annotation) error
	NotifyAnnotationReopened(ctx context.Context, a *annotation.Annotation) error
	NotifyDueApproaching(ctx context.Context, a *annotation.Annotation, when time.Time) error
}

// AuditLogger abstracts the audit writer from the application layer.
type AuditLogger interface {
	Log(ctx context.Context, action, actorID, entityType, entityID, before, after string) error
}

// ClockAdapter wraps a Clock for the standard library.
type TimeProvider interface {
	Now() time.Time
}
