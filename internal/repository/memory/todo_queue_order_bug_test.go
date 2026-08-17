package memory_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/cry048/design-review-platform/internal/domain/annotation"
	"github.com/cry048/design-review-platform/internal/domain/canvas"
	"github.com/cry048/design-review-platform/internal/repository/memory"
	"github.com/cry048/design-review-platform/internal/service"
)

type todoClock struct {
	now time.Time
}

func (c todoClock) Now() time.Time { return c.now }

func todoAnnotation(t *testing.T, id, assignee string, priority annotation.Priority, status annotation.Status, created time.Time, due *time.Time) *annotation.Annotation {
	t.Helper()
	anchor := canvas.NewPointAnchor("version-1", canvas.Coordinate{X: 12, Y: 24})
	item, err := annotation.NewAnnotation(id, "project-1", "board-1", "version-1", id, "待办排序测试", "reporter-1", anchor, priority, created)
	if err != nil {
		t.Fatalf("create annotation %s: %v", id, err)
	}
	item.AssigneeID = assignee
	item.Status = status
	item.DueAt = due
	return item
}

func TestPersonalTodoQueueOrdersUrgencyDeterministically(t *testing.T) {
	now := time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC)
	overdueOld := now.Add(-48 * time.Hour)
	overdueRecent := now.Add(-time.Hour)
	upcomingSoon := now.Add(time.Hour)
	upcomingLater := now.Add(8 * time.Hour)

	repo := memory.NewAnnotationRepo()
	items := []*annotation.Annotation{
		todoAnnotation(t, "unscheduled-low", "reviewer-1", annotation.PriorityLow, annotation.StatusOpen, now.Add(-72*time.Hour), nil),
		todoAnnotation(t, "upcoming-later", "reviewer-1", annotation.PriorityHigh, annotation.StatusReview, now.Add(-60*time.Hour), &upcomingLater),
		todoAnnotation(t, "overdue-recent", "reviewer-1", annotation.PriorityCritical, annotation.StatusReplied, now.Add(-36*time.Hour), &overdueRecent),
		todoAnnotation(t, "overdue-old", "reviewer-1", annotation.PriorityLow, annotation.StatusOpen, now.Add(-24*time.Hour), &overdueOld),
		todoAnnotation(t, "upcoming-soon", "reviewer-1", annotation.PriorityNormal, annotation.StatusReview, now.Add(-12*time.Hour), &upcomingSoon),
		todoAnnotation(t, "unscheduled-critical", "reviewer-1", annotation.PriorityCritical, annotation.StatusReplied, now.Add(-6*time.Hour), nil),
		todoAnnotation(t, "resolved", "reviewer-1", annotation.PriorityCritical, annotation.StatusResolved, now.Add(-3*time.Hour), &overdueOld),
		todoAnnotation(t, "closed", "reviewer-1", annotation.PriorityCritical, annotation.StatusClosed, now.Add(-2*time.Hour), &overdueOld),
		todoAnnotation(t, "another-reviewer", "reviewer-2", annotation.PriorityCritical, annotation.StatusOpen, now.Add(-time.Hour), &overdueOld),
	}
	for _, item := range items {
		if err := repo.Save(context.Background(), item); err != nil {
			t.Fatalf("save %s: %v", item.ID, err)
		}
	}

	svc := service.AnnotationService{Annotations: repo, Clock: todoClock{now: now}}
	want := []string{
		"overdue-old",
		"overdue-recent",
		"upcoming-soon",
		"upcoming-later",
		"unscheduled-critical",
		"unscheduled-low",
	}
	for attempt := 0; attempt < 5; attempt++ {
		got, err := svc.ListForUser(context.Background(), "reviewer-1")
		if err != nil {
			t.Fatalf("list personal todos: %v", err)
		}
		ids := make([]string, len(got))
		for i, item := range got {
			ids[i] = item.ID
		}
		if !reflect.DeepEqual(ids, want) {
			t.Fatalf("attempt %d todo order = %v, want %v", attempt+1, ids, want)
		}
	}
}
