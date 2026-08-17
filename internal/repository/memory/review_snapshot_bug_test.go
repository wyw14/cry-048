package memory

import (
	"context"
	"testing"
	"time"

	"github.com/cry048/design-review-platform/internal/domain/review"
)

func TestReviewRepositoryGetDoesNotMutateFrozenSnapshot(t *testing.T) {
	repo := NewReviewRepo()
	round, err := review.NewRound("round-iso", "project-iso", "board-iso", "first review", time.Now().UTC())
	if err != nil { t.Fatal(err) }
	if err := round.AddSnapshot(review.Snapshot{ID: "snapshot-iso", Counts: map[string]int{"open": 2}}, time.Now().UTC()); err != nil { t.Fatal(err) }
	if err := repo.SaveRound(context.Background(), round); err != nil { t.Fatal(err) }
	got, err := repo.GetRound(context.Background(), round.ID)
	if err != nil { t.Fatal(err) }
	got.Snapshots[0].Counts["open"] = 99
	got.Snapshots[0].Counts["closed"] = 4
	again, err := repo.GetRound(context.Background(), round.ID)
	if err != nil { t.Fatal(err) }
	if again.Snapshots[0].Counts["open"] != 2 { t.Fatalf("snapshot count leaked through repository: %#v", again.Snapshots[0].Counts) }
	if _, ok := again.Snapshots[0].Counts["closed"]; ok { t.Fatal("caller-added snapshot count leaked into frozen round") }
}
