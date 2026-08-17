package annotation

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cry048/design-review-platform/internal/domain/canvas"
)

func fixedTime() time.Time {
	return time.Date(2026, 8, 17, 10, 0, 0, 0, time.UTC)
}

func TestPriorityValid(t *testing.T) {
	cases := []struct {
		p    Priority
		want bool
	}{
		{PriorityLow, true},
		{PriorityNormal, true},
		{PriorityHigh, true},
		{PriorityCritical, true},
		{Priority("xyz"), false},
		{Priority(""), false},
	}
	for _, c := range cases {
		if got := c.p.Valid(); got != c.want {
			t.Errorf("Priority(%q).Valid=%v want %v", c.p, got, c.want)
		}
	}
}

func TestStatusTransitionTable(t *testing.T) {
	allowed := map[Status][]Status{
		StatusOpen:     {StatusReplied, StatusReview, StatusClosed},
		StatusReplied:  {StatusReview, StatusOpen, StatusClosed},
		StatusReview:   {StatusResolved, StatusOpen, StatusClosed},
		StatusResolved: {StatusOpen},
		StatusClosed:   {},
	}
	for from, tos := range allowed {
		for _, to := range tos {
			if !from.CanTransitionTo(to) {
				t.Errorf("%s -> %s should be allowed", from, to)
			}
		}
	}
	// negative cases
	disallowed := []struct {
		from Status
		to   Status
	}{
		{StatusOpen, StatusResolved},
		{StatusReplied, StatusResolved},
		{StatusResolved, StatusClosed},
		{StatusResolved, StatusReplied},
		{StatusClosed, StatusOpen},
		{StatusClosed, StatusReplied},
		{StatusClosed, StatusResolved},
		{StatusClosed, StatusClosed},
	}
	for _, c := range disallowed {
		if c.from.CanTransitionTo(c.to) {
			t.Errorf("%s -> %s should be DISALLOWED", c.from, c.to)
		}
	}
}

func TestNewAnnotationValidation(t *testing.T) {
	now := fixedTime()
	anchor := canvas.NewPointAnchor("v1", canvas.Coordinate{X: 10, Y: 10})
	cases := []struct {
		name string
		args map[string]string
	}{
		{"missing id", map[string]string{"id": "", "project_id": "p", "board_id": "b", "version_id": "v", "title": "t", "reporter_id": "r"}},
		{"missing title", map[string]string{"id": "a", "project_id": "p", "board_id": "b", "version_id": "v", "title": "", "reporter_id": "r"}},
		{"missing reporter", map[string]string{"id": "a", "project_id": "p", "board_id": "b", "version_id": "v", "title": "t", "reporter_id": ""}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := NewAnnotation(c.args["id"], c.args["project_id"], c.args["board_id"], c.args["version_id"], c.args["title"], "", c.args["reporter_id"], anchor, PriorityNormal, now)
			if !errors.Is(err, ErrMustNotBeEmpty) {
				t.Fatalf("want ErrMustNotBeEmpty, got %v", err)
			}
		})
	}
	// invalid priority
	if _, err := NewAnnotation("a", "p", "b", "v", "t", "", "r", anchor, Priority("xyz"), now); !errors.Is(err, ErrInvalidPriority) {
		t.Fatalf("want ErrInvalidPriority, got %v", err)
	}
}

func TestAnnotationHappyPathResolve(t *testing.T) {
	now := fixedTime()
	anchor := canvas.NewPointAnchor("v1", canvas.Coordinate{X: 5, Y: 5})
	a, err := NewAnnotation("a1", "p", "b", "v1", "title", "body", "reporter", anchor, PriorityNormal, now)
	if err != nil {
		t.Fatal(err)
	}
	if a.Status != StatusOpen {
		t.Fatalf("new annotation status=%s want open", a.Status)
	}
	// reply -> open -> replied
	if _, err := a.AddReply("r1", "designer-1", "已修复", now); err != nil {
		t.Fatal(err)
	}
	if a.Status != StatusReplied {
		t.Fatalf("after reply status=%s want replied", a.Status)
	}
	// request review
	if err := a.RequestReview(now, "designer-1"); err != nil {
		t.Fatal(err)
	}
	if a.Status != StatusReview {
		t.Fatalf("after review status=%s want review", a.Status)
	}
	// resolve
	if err := a.Resolve(now, "reviewer-1"); err != nil {
		t.Fatal(err)
	}
	if a.Status != StatusResolved {
		t.Fatalf("after resolve status=%s want resolved", a.Status)
	}
	if a.ResolvedBy != "reviewer-1" {
		t.Fatalf("resolved_by=%s want reviewer-1", a.ResolvedBy)
	}
	if a.ResolvedAt == nil {
		t.Fatal("resolved_at should be set")
	}
}

func TestResolveWithoutReplyFails(t *testing.T) {
	now := fixedTime()
	anchor := canvas.NewPointAnchor("v1", canvas.Coordinate{X: 5, Y: 5})
	a, _ := NewAnnotation("a1", "p", "b", "v1", "title", "body", "reporter", anchor, PriorityNormal, now)
	// attempt direct review without reply
	if err := a.RequestReview(now, "designer-1"); err != nil {
		t.Fatal(err)
	}
	if err := a.Resolve(now, "reviewer-1"); !errors.Is(err, ErrCannotResolveWithoutReply) {
		t.Fatalf("want ErrCannotResolveWithoutReply, got %v", err)
	}
}

func TestReopenOnlyFromResolved(t *testing.T) {
	now := fixedTime()
	anchor := canvas.NewPointAnchor("v1", canvas.Coordinate{X: 5, Y: 5})
	a, _ := NewAnnotation("a1", "p", "b", "v1", "title", "body", "reporter", anchor, PriorityNormal, now)
	// cannot reopen from open
	if err := a.Reopen(now, "reviewer-1", "复核"); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("want ErrInvalidTransition, got %v", err)
	}
	// happy path to resolved
	if _, err := a.AddReply("r1", "d", "ok", now); err != nil {
		t.Fatal(err)
	}
	if err := a.RequestReview(now, "d"); err != nil {
		t.Fatal(err)
	}
	if err := a.Resolve(now, "rv"); err != nil {
		t.Fatal(err)
	}
	// reopen must succeed
	if err := a.Reopen(now, "rv", "复核重新打开"); err != nil {
		t.Fatalf("reopen failed: %v", err)
	}
	if a.Status != StatusOpen {
		t.Fatalf("status=%s want open", a.Status)
	}
	if a.ResolvedAt != nil {
		t.Fatal("resolved_at should be cleared")
	}
}

func TestClosedIsTerminal(t *testing.T) {
	now := fixedTime()
	anchor := canvas.NewPointAnchor("v1", canvas.Coordinate{X: 5, Y: 5})
	a, _ := NewAnnotation("a1", "p", "b", "v1", "title", "body", "reporter", anchor, PriorityNormal, now)
	if err := a.Close(now, "actor"); err != nil {
		t.Fatal(err)
	}
	// cannot transition out of closed
	if err := a.Reopen(now, "rv", ""); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("want ErrInvalidTransition, got %v", err)
	}
	if err := a.RequestReview(now, "d"); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("want ErrInvalidTransition, got %v", err)
	}
}

func TestDueAtPastRejected(t *testing.T) {
	now := fixedTime()
	past := now.Add(-1 * time.Hour)
	anchor := canvas.NewPointAnchor("v1", canvas.Coordinate{X: 5, Y: 5})
	a, _ := NewAnnotation("a1", "p", "b", "v1", "t", "", "r", anchor, PriorityNormal, now)
	if err := a.SetDueAt(&past, now); !errors.Is(err, ErrDueDateInPast) {
		t.Fatalf("want ErrDueDateInPast, got %v", err)
	}
	future := now.Add(48 * time.Hour)
	if err := a.SetDueAt(&future, now); err != nil {
		t.Fatalf("future due ok: %v", err)
	}
	if a.DueAt == nil || !a.DueAt.Equal(future) {
		t.Fatalf("due_at not set correctly: %v", a.DueAt)
	}
	// clear
	if err := a.SetDueAt(nil, now); err != nil {
		t.Fatalf("clear due: %v", err)
	}
	if a.DueAt != nil {
		t.Fatal("due_at should be nil")
	}
}

func TestAddAttachment(t *testing.T) {
	now := fixedTime()
	anchor := canvas.NewPointAnchor("v1", canvas.Coordinate{X: 5, Y: 5})
	a, _ := NewAnnotation("a1", "p", "b", "v1", "t", "", "r", anchor, PriorityNormal, now)
	att := Attachment{ID: "att1", Filename: "x.png", MediaType: "image/png", Size: 1000, StorageKey: "k1", UploadedBy: "u"}
	a.AddAttachment(att, now)
	if len(a.Attachments) != 1 {
		t.Fatalf("attachments=%d want 1", len(a.Attachments))
	}
	if err := a.RemoveAttachment("att1", now); err != nil {
		t.Fatal(err)
	}
	if len(a.Attachments) != 0 {
		t.Fatalf("attachments=%d want 0", len(a.Attachments))
	}
	if err := a.RemoveAttachment("nonexistent", now); !errors.Is(err, ErrAttachmentNotFound) {
		t.Fatalf("want ErrAttachmentNotFound, got %v", err)
	}
}

func TestMigrateAnchor(t *testing.T) {
	now := fixedTime()
	anchor := canvas.NewPointAnchor("v1", canvas.Coordinate{X: 5, Y: 5})
	a, _ := NewAnnotation("a1", "p", "b", "v1", "t", "", "r", anchor, PriorityNormal, now)
	newAnchor := canvas.NewPointAnchor("v2", canvas.Coordinate{X: 10, Y: 10})
	m, err := a.MigrateAnchor("v2", newAnchor, "版本升级", "designer-1", now)
	if err != nil {
		t.Fatal(err)
	}
	if m.FromVersionID != "v1" || m.ToVersionID != "v2" {
		t.Fatalf("migration record incorrect: %+v", m)
	}
	if a.VersionID != "v2" {
		t.Fatalf("annotation version_id=%s want v2", a.VersionID)
	}
	if a.Anchor.Point.X != 10 || a.Anchor.Point.Y != 10 {
		t.Fatalf("anchor not updated: %+v", a.Anchor.Point)
	}
}

func TestMarkStale(t *testing.T) {
	now := fixedTime()
	anchor := canvas.NewPointAnchor("v1", canvas.Coordinate{X: 5, Y: 5})
	a, _ := NewAnnotation("a1", "p", "b", "v1", "t", "original body", "r", anchor, PriorityNormal, now)
	a.MarkStale("版本已替代", now)
	if !strings.HasPrefix(a.Body, "[已失效") {
		t.Fatalf("body should be prefixed with stale marker, got: %s", a.Body)
	}
	// idempotent
	prev := a.Body
	a.MarkStale("版本已替代", now)
	if a.Body != prev {
		t.Fatalf("body should not change twice: %q vs %q", a.Body, prev)
	}
}
