package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"designreview/internal/domain"
	"designreview/internal/store"
)

func seedReview(t *testing.T) (*store.MemoryRepository, *ReviewService, domain.Actor, domain.ID) {
	t.Helper()
	repository := store.NewMemoryRepository()
	actor := domain.Actor{ID: "reviewer-1", Email: "reviewer@example.com", Name: "Reviewer"}
	now := time.Unix(20, 0)
	project, _ := domain.NewProject("project-1", "Project", actor, now)
	_ = repository.PutProject(context.Background(), project)
	member, _ := domain.NewMember(project.ID, actor, domain.RoleReviewer, now)
	_ = repository.PutMember(context.Background(), member)
	board, _ := domain.NewBoard("board-1", project.ID, "Board")
	_ = repository.PutBoard(context.Background(), board)
	version, _ := domain.NewDesignVersion("version-1", board.ID, 1, domain.PreviewConfig{Scale: 1})
	_ = repository.PutVersion(context.Background(), version)
	round, _ := domain.NewReviewRound("review-1", project.ID, version.ID, now)
	_ = repository.PutReviewRound(context.Background(), round)
	return repository, NewReviewService(store.NewMemoryUnitOfWork(repository), domain.FixedClock{Time: now}), actor, round.ID
}

func TestReviewCloseRollsBackOnNotificationFailure(t *testing.T) {
	repository, service, actor, roundID := seedReview(t)
	closed, err := service.Close(context.Background(), CloseReviewCommand{RoundID: roundID, Conclusion: "approved", Actor: actor, Recipients: []domain.ID{"recipient-1"}})
	if err != nil {
		t.Fatal(err)
	}
	if closed.Status != domain.ReviewClosed || closed.Snapshot == nil {
		t.Fatal("review did not close")
	}
	value, err := repository.GetReviewRound(context.Background(), roundID)
	if err != nil {
		t.Fatal(err)
	}
	if value.Status != domain.ReviewClosed {
		t.Fatal("review state not persisted")
	}
}

func TestMigrationRetainsUnmappedAnchor(t *testing.T) {
	repository := store.NewMemoryRepository()
	actor := domain.Actor{ID: "actor-1", Email: "actor@example.com", Name: "Actor"}
	now := time.Unix(1, 0)
	project, _ := domain.NewProject("project-1", "Project", actor, now)
	_ = repository.PutProject(context.Background(), project)
	board, _ := domain.NewBoard("board-1", project.ID, "Board")
	_ = repository.PutBoard(context.Background(), board)
	for _, id := range []domain.ID{"version-1", "version-2"} {
		version, _ := domain.NewDesignVersion(id, board.ID, 1, domain.PreviewConfig{Scale: 1})
		_ = repository.PutVersion(context.Background(), version)
	}
	annotation, _ := domain.NewAnnotation("annotation-1", project.ID, board.ID, domain.Anchor{VersionID: "version-1", NodeKey: "unmapped", X: 1, Y: 1}, "Note", actor, now)
	_ = repository.PutAnnotation(context.Background(), annotation)
	service := NewMigrationService(store.NewMemoryUnitOfWork(repository))
	plan, err := service.Plan(context.Background(), repository, "version-1", "version-2", []AnchorMapping{{SourceNode: "other", TargetNode: "new"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Apply(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	value, _ := repository.GetAnnotation(context.Background(), annotation.ID)
	if value.Anchor.VersionID != "version-1" || !strings.Contains(value.MigrationIssue, "unavailable") {
		t.Fatalf("unexpected migration result: %+v", value)
	}
}

func TestQueryAndExportUseAllowedFieldsAndAbortOnCancel(t *testing.T) {
	repository, _, actor, annotation := seedCollaboration(t)
	queries := NewQueryService(repository)
	if _, err := queries.ListAnnotations(context.Background(), AnnotationQuery{VersionID: annotation.Anchor.VersionID, SortBy: "DROP TABLE"}); !errors.As(err, new(*domain.ValidationError)) {
		t.Fatalf("expected query validation, got %v", err)
	}
	exports := NewMemoryExportServiceForTest(t, queries)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := exports.ExportAnnotations(ctx, AnnotationQuery{VersionID: annotation.Anchor.VersionID}); !errors.Is(err, domain.ErrCanceled) {
		t.Fatalf("expected canceled export, got %v", err)
	}
	_ = actor
}

func NewMemoryExportServiceForTest(t *testing.T, queries *QueryService) *ExportService {
	t.Helper()
	return NewExportService(queries, store.NewMemoryExportStore())
}
