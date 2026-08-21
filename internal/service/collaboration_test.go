package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"designreview/internal/domain"
	"designreview/internal/store"
)

func seedCollaboration(t *testing.T) (*store.MemoryRepository, *CollaborationService, domain.Actor, domain.Annotation) {
	t.Helper()
	repository := store.NewMemoryRepository()
	actor := domain.Actor{ID: "actor-1", Email: "designer@example.com", Name: "Designer"}
	clock := domain.FixedClock{Time: time.Unix(100, 0)}
	project, err := domain.NewProject("project-1", "Offline board", actor, clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.PutProject(context.Background(), project); err != nil {
		t.Fatal(err)
	}
	member, err := domain.NewMember(project.ID, actor, domain.RoleEditor, clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.PutMember(context.Background(), member); err != nil {
		t.Fatal(err)
	}
	board, err := domain.NewBoard("board-1", project.ID, "Main")
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.PutBoard(context.Background(), board); err != nil {
		t.Fatal(err)
	}
	annotation, err := domain.NewAnnotation("annotation-1", project.ID, board.ID, domain.Anchor{VersionID: "version-1", NodeKey: "hero", X: 1, Y: 2}, "Original", actor, clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.PutAnnotation(context.Background(), annotation); err != nil {
		t.Fatal(err)
	}
	return repository, NewCollaborationService(repository, clock), actor, annotation
}

func TestCollaborationExpectedVersionAllowsOneConcurrentWriter(t *testing.T) {
	repository, service, actor, annotation := seedCollaboration(t)
	var group sync.WaitGroup
	results := make(chan error, 2)
	for _, body := range []string{"one", "two"} {
		group.Add(1)
		go func(body string) {
			defer group.Done()
			_, err := service.EditAnnotation(context.Background(), EditAnnotationCommand{AnnotationID: annotation.ID, Expected: annotation.Revision, Body: body, Actor: actor})
			results <- err
		}(body)
	}
	group.Wait()
	close(results)
	success := 0
	conflicts := 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, domain.ErrConflict) {
			conflicts++
		} else {
			t.Errorf("unexpected error: %v", err)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatalf("success=%d conflicts=%d", success, conflicts)
	}
	value, err := repository.GetAnnotation(context.Background(), annotation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if value.Revision != 2 || len(value.History) != 2 {
		t.Fatalf("revision=%d history=%d", value.Revision, len(value.History))
	}
}

func TestMembershipNormalizesAndRejectsDuplicateEmail(t *testing.T) {
	repository := store.NewMemoryRepository()
	clock := domain.FixedClock{Time: time.Unix(1, 0)}
	service := NewMembershipService(repository, clock)
	actor := domain.Actor{ID: "actor-1", Email: " Person@Example.com ", Name: "Person"}
	if _, err := service.Add(context.Background(), "project-1", actor, domain.RoleReviewer); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Add(context.Background(), domain.ID("project-1"), domain.Actor{ID: "actor-2", Email: "person@example.com", Name: "Other"}, domain.RoleViewer); !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("expected duplicate rejection, got %v", err)
	}
}
