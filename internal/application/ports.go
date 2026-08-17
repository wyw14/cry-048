// Package application defines use-case interfaces and DTOs.
package application

import (
	"context"
	"time"

	"github.com/cry048/design-review-platform/internal/domain/annotation"
	"github.com/cry048/design-review-platform/internal/domain/canvas"
	"github.com/cry048/design-review-platform/internal/domain/project"
	"github.com/cry048/design-review-platform/internal/domain/review"
)

// ProjectRepository persists projects, boards, versions, and memberships.
type ProjectRepository interface {
	SaveProject(ctx context.Context, p *project.Project) error
	GetProject(ctx context.Context, id string) (*project.Project, error)
	ListProjects(ctx context.Context, filter ProjectFilter) ([]*project.Project, int, error)
	ArchiveProject(ctx context.Context, id string, version int) error

	SaveBoard(ctx context.Context, b *project.Board) error
	GetBoard(ctx context.Context, id string) (*project.Board, error)
	ListBoards(ctx context.Context, projectID string) ([]*project.Board, error)
	UpdateBoard(ctx context.Context, b *project.Board) error
	CloseBoard(ctx context.Context, id string, version int) error

	SaveVersion(ctx context.Context, v *project.Version) error
	GetVersion(ctx context.Context, id string) (*project.Version, error)
	ListVersions(ctx context.Context, boardID string) ([]*project.Version, error)
	LatestVersion(ctx context.Context, boardID string) (*project.Version, error)
	UpdateVersion(ctx context.Context, v *project.Version) error

	SaveMembership(ctx context.Context, m *project.Membership) error
	GetMembership(ctx context.Context, projectID, userID string) (*project.Membership, error)
	ListMemberships(ctx context.Context, projectID string) ([]*project.Membership, error)
	DeleteMembership(ctx context.Context, id string) error
}

// ProjectFilter restricts a project listing.
type ProjectFilter struct {
	Status string
	Name   string
}

// AnnotationRepository persists annotations and their sub-entities.
type AnnotationRepository interface {
	Save(ctx context.Context, a *annotation.Annotation) error
	Get(ctx context.Context, id string) (*annotation.Annotation, error)
	Update(ctx context.Context, a *annotation.Annotation, expectedVersion int) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter AnnotationFilter) ([]*annotation.Annotation, int, error)
	Search(ctx context.Context, q string, limit int) ([]*annotation.Annotation, error)
	ListTodos(ctx context.Context, query TodoQuery) ([]*annotation.Annotation, error)
	BulkMigrateAnchors(ctx context.Context, fromVersionID, toVersionID, by, reason string, now time.Time) (int, error)
}

// AnnotationFilter restricts a listing.
type AnnotationFilter struct {
	ProjectID  string
	BoardID    string
	VersionID  string
	Status     string
	Priority   string
	AssigneeID string
	ReporterID string
	From       time.Time
	To         time.Time
	Page       int
	PageSize   int
	SortBy     string
	SortDesc   bool
}

// TodoQuery defines the stable personal-work ordering boundary.
// Now comes from the application clock so overdue classification is repeatable.
type TodoQuery struct {
	AssigneeID string
	Now        time.Time
	Limit      int
}

// ReviewRepository persists review rounds and snapshots.
type ReviewRepository interface {
	SaveRound(ctx context.Context, r *review.Round) error
	GetRound(ctx context.Context, id string) (*review.Round, error)
	ListRounds(ctx context.Context, boardID string) ([]*review.Round, error)
	UpdateRound(ctx context.Context, r *review.Round, expectedVersion int) error
	SaveSnapshot(ctx context.Context, s review.Snapshot) error
	ListSnapshots(ctx context.Context, roundID string) ([]review.Snapshot, error)
}

// Clock supplies a current time.
type Clock interface {
	Now() time.Time
}

// IDGenerator generates unique identifiers.
type IDGenerator interface {
	New() string
}

// CanvasValidator checks that an anchor lies within a version's board size.
type CanvasValidator interface {
	ValidateAnchor(ctx context.Context, boardID string, v project.Version, anchor canvas.Anchor) error
}

// TxRunner runs fn inside a transaction (or no-op for in-memory).
type TxRunner interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}
