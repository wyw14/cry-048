package memory

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/cry048/design-review-platform/internal/application"
	"github.com/cry048/design-review-platform/internal/domain/annotation"
	"github.com/cry048/design-review-platform/internal/domain/canvas"
	"github.com/cry048/design-review-platform/internal/domain/project"
)

func TestProjectRepoCRUD(t *testing.T) {
	ctx := context.Background()
	r := NewProjectRepo()
	now := time.Now()
	p, _ := project.NewProject("p1", "name", "", now)
	if err := r.SaveProject(ctx, p); err != nil {
		t.Fatal(err)
	}
	got, err := r.GetProject(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "name" {
		t.Fatalf("name=%s", got.Name)
	}
	// not found
	if _, err := r.GetProject(ctx, "missing"); !errors.Is(err, project.ErrProjectNotFound) {
		t.Fatalf("want ErrProjectNotFound, got %v", err)
	}
}

func TestVersionUniqueConstraint(t *testing.T) {
	ctx := context.Background()
	r := NewProjectRepo()
	now := time.Now()
	p, _ := project.NewProject("p1", "n", "", now)
	_ = r.SaveProject(ctx, p)
	b, _ := project.NewBoard("b1", "p1", "board", project.BoardSize{Width: 100, Height: 100}, now)
	_ = r.SaveBoard(ctx, b)
	v1, _ := project.NewVersion("v1", "b1", 1, "label", "key", "", "u", now)
	if err := r.SaveVersion(ctx, v1); err != nil {
		t.Fatal(err)
	}
	v1dup, _ := project.NewVersion("v1-dup", "b1", 1, "label2", "key2", "", "u", now)
	if err := r.SaveVersion(ctx, v1dup); !errors.Is(err, project.ErrVersionAlreadyExists) {
		t.Fatalf("want ErrVersionAlreadyExists, got %v", err)
	}
}

func TestMembershipUnique(t *testing.T) {
	ctx := context.Background()
	r := NewProjectRepo()
	now := time.Now()
	p, _ := project.NewProject("p1", "n", "", now)
	_ = r.SaveProject(ctx, p)
	m1, _ := project.NewMembership("m1", "p1", "u1", project.RoleEditor, now)
	if err := r.SaveMembership(ctx, m1); err != nil {
		t.Fatal(err)
	}
	m2, _ := project.NewMembership("m2", "p1", "u1", project.RoleViewer, now)
	if err := r.SaveMembership(ctx, m2); !errors.Is(err, project.ErrMemberAlreadyExists) {
		t.Fatalf("want ErrMemberAlreadyExists, got %v", err)
	}
}

func TestAnnotationOptimisticLock(t *testing.T) {
	ctx := context.Background()
	r := NewAnnotationRepo()
	now := time.Now()
	anchor := canvas.NewPointAnchor("v1", canvas.Coordinate{X: 5, Y: 5})
	a, _ := annotation.NewAnnotation("a1", "p", "b", "v1", "title", "", "r", anchor, annotation.PriorityNormal, now)
	_ = r.Save(ctx, a)
	// read fresh
	a1, _ := r.Get(ctx, "a1")
	if a1.Version != 1 {
		t.Fatalf("version=%d want 1", a1.Version)
	}
	// simulate stale update
	stale := *a1
	stale.Version = 99
	if err := r.Update(ctx, &stale, 99); !errors.Is(err, annotation.ErrStaleVersion) {
		t.Fatalf("want ErrStaleVersion, got %v", err)
	}
	// correct update with expected version
	a1.Priority = annotation.PriorityHigh
	a1.Version = 2
	if err := r.Update(ctx, a1, 1); err != nil {
		t.Fatalf("update failed: %v", err)
	}
}

func TestAnnotationListFiltering(t *testing.T) {
	ctx := context.Background()
	r := NewAnnotationRepo()
	now := time.Now()
	anchor := canvas.NewPointAnchor("v1", canvas.Coordinate{X: 5, Y: 5})
	a1, _ := annotation.NewAnnotation("a1", "p1", "b1", "v1", "t1", "", "u1", anchor, annotation.PriorityHigh, now)
	a2, _ := annotation.NewAnnotation("a2", "p1", "b1", "v1", "t2", "", "u2", anchor, annotation.PriorityNormal, now)
	a2.AssigneeID = "u1"
	_ = r.Save(ctx, a1)
	_ = r.Save(ctx, a2)

	// filter by assignee
	got, _, err := r.List(ctx, application.AnnotationFilter{AssigneeID: "u1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "a2" {
		t.Fatalf("unexpected list: %+v", got)
	}
}

func TestAnnotationSearch(t *testing.T) {
	ctx := context.Background()
	r := NewAnnotationRepo()
	now := time.Now()
	anchor := canvas.NewPointAnchor("v1", canvas.Coordinate{X: 5, Y: 5})
	a1, _ := annotation.NewAnnotation("a1", "p", "b", "v", "对比度不足", "白色文字", "r", anchor, annotation.PriorityNormal, now)
	a2, _ := annotation.NewAnnotation("a2", "p", "b", "v", "按钮样式", "主操作", "r", anchor, annotation.PriorityNormal, now)
	_ = r.Save(ctx, a1)
	_ = r.Save(ctx, a2)

	results, err := r.Search(ctx, "对比度", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].ID != "a1" {
		t.Fatalf("search results: %+v", results)
	}
	// also matches body
	results, _ = r.Search(ctx, "主操作", 10)
	if len(results) != 1 || results[0].ID != "a2" {
		t.Fatalf("search body results: %+v", results)
	}
}

func TestBulkMigrateAnchors(t *testing.T) {
	ctx := context.Background()
	r := NewAnnotationRepo()
	now := time.Now()
	anchor := canvas.NewPointAnchor("v1", canvas.Coordinate{X: 5, Y: 5})
	a1, _ := annotation.NewAnnotation("a1", "p", "b", "v1", "t1", "", "r", anchor, annotation.PriorityNormal, now)
	a2, _ := annotation.NewAnnotation("a2", "p", "b", "v1", "t2", "", "r", anchor, annotation.PriorityNormal, now)
	a3, _ := annotation.NewAnnotation("a3", "p", "b", "v2", "t3", "", "r", anchor, annotation.PriorityNormal, now)
	_ = r.Save(ctx, a1)
	_ = r.Save(ctx, a2)
	_ = r.Save(ctx, a3)

	n, err := r.BulkMigrateAnchors(ctx, "v1", "v2", "u", "test", now)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("migrated=%d want 2", n)
	}
	// verify the annotations now point to v2
	for _, id := range []string{"a1", "a2"} {
		a, _ := r.Get(ctx, id)
		if a.VersionID != "v2" {
			t.Fatalf("annotation %s version_id=%s want v2", id, a.VersionID)
		}
	}
}

func TestConcurrencyAppendSafe(t *testing.T) {
	ctx := context.Background()
	r := NewAnnotationRepo()
	var wg sync.WaitGroup
	now := time.Now()
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			anchor := canvas.NewPointAnchor("v", canvas.Coordinate{X: 1, Y: 1})
			a, _ := annotation.NewAnnotation("a"+itoa(i), "p", "b", "v", "t", "", "r", anchor, annotation.PriorityNormal, now)
			_ = r.Save(ctx, a)
		}(i)
	}
	wg.Wait()
	list, total, _ := r.List(ctx, application.AnnotationFilter{PageSize: 1000})
	if total != 50 {
		t.Fatalf("total=%d want 50", total)
	}
	if len(list) != 50 {
		t.Fatalf("len=%d want 50", len(list))
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}
