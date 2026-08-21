package domain

import (
	"errors"
	"testing"
	"time"
)

func testActor() Actor { return Actor{ID: "actor-1", Email: "designer@example.com", Name: "Designer"} }

func TestAnnotationStateMachinePreservesHistory(t *testing.T) {
	actor := testActor()
	annotation, err := NewAnnotation("annotation-1", "project-1", "board-1", Anchor{VersionID: "version-1", NodeKey: "hero", X: 10, Y: 20}, "Initial note", actor, time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	annotation, err = annotation.Transition(AnnotationReview, actor, time.Unix(2, 0))
	if err != nil {
		t.Fatal(err)
	}
	annotation, err = annotation.Edit("Updated note", actor, time.Unix(3, 0))
	if err != nil {
		t.Fatal(err)
	}
	annotation, err = annotation.Transition(AnnotationResolved, actor, time.Unix(4, 0))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := annotation.Edit("forbidden", actor, time.Now()); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected resolved edit rejection, got %v", err)
	}
	annotation, err = annotation.Transition(AnnotationOpen, actor, time.Unix(5, 0))
	if err != nil {
		t.Fatal(err)
	}
	if annotation.State != AnnotationOpen || len(annotation.History) != 5 {
		t.Fatalf("state=%s history=%d", annotation.State, len(annotation.History))
	}
}

func TestVersionPublishSupersedesWithoutPreviewAliasing(t *testing.T) {
	version, err := NewDesignVersion("version-2", "board-1", 2, PreviewConfig{Background: "white", Scale: 1.5, Hotspots: []string{"hero"}})
	if err != nil {
		t.Fatal(err)
	}
	published, err := version.Publish(time.Unix(10, 0))
	if err != nil {
		t.Fatal(err)
	}
	old, err := published.Supersede("version-3")
	if err != nil {
		t.Fatal(err)
	}
	old.Preview.Hotspots[0] = "changed"
	if published.Preview.Hotspots[0] != "hero" {
		t.Fatal("preview configuration was aliased")
	}
}
