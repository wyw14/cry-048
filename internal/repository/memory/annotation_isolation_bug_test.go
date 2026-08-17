package memory

import (
	"context"
	"testing"
	"time"

	"github.com/cry048/design-review-platform/internal/domain/annotation"
	"github.com/cry048/design-review-platform/internal/domain/canvas"
)

func TestAnnotationRepositoryGetReturnsIsolatedAggregate(t *testing.T) {
	repo := NewAnnotationRepo()
	now := time.Now().UTC()
	a, err := annotation.NewAnnotation("a-iso", "p-iso", "b-iso", "v-iso", "title", "body", "reporter", canvas.NewPointAnchor("v-iso", canvas.Coordinate{X: 4, Y: 8}), annotation.PriorityHigh, now)
	if err != nil { t.Fatal(err) }
	if _, err := a.AddReply("r-iso", "author", "original", now); err != nil { t.Fatal(err) }
	if err := repo.Save(context.Background(), a); err != nil { t.Fatal(err) }
	got, err := repo.Get(context.Background(), a.ID)
	if err != nil { t.Fatal(err) }
	got.Replies[0].Body = "tampered"
	got.Anchor.Point.X = 99
	again, err := repo.Get(context.Background(), a.ID)
	if err != nil { t.Fatal(err) }
	if again.Replies[0].Body != "original" { t.Fatalf("repository reply storage was aliased: %q", again.Replies[0].Body) }
	if again.Anchor.Point.X != 4 { t.Fatalf("repository anchor storage was aliased: %v", again.Anchor.Point.X) }
}
