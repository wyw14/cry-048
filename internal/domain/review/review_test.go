package review

import (
	"errors"
	"testing"
	"time"
)

func TestRoundLifecycle(t *testing.T) {
	now := time.Now()
	r, _ := NewRound("r1", "p1", "b1", "标题", now)
	if r.Status != "open" {
		t.Fatalf("initial status=%s want open", r.Status)
	}
	snap := Snapshot{ID: "s1", ProjectID: "p1", BoardID: "b1", VersionID: "v1", CreatedBy: "u"}
	if err := r.AddSnapshot(snap, now); err != nil {
		t.Fatal(err)
	}
	if len(r.Snapshots) != 1 {
		t.Fatalf("snapshots=%d want 1", len(r.Snapshots))
	}
	if err := r.SetConclusion("ok", RecApprove, "reviewer", now); err != nil {
		t.Fatal(err)
	}
	if r.Conclusion != "ok" || r.Recommendation != RecApprove {
		t.Fatalf("conclusion not set: %+v", r)
	}
	if err := r.Close(now); err != nil {
		t.Fatal(err)
	}
	if r.Status != "closed" {
		t.Fatalf("status=%s want closed", r.Status)
	}
	// operations on closed round should fail
	if err := r.AddSnapshot(snap, now); !errors.Is(err, ErrRoundClosed) {
		t.Fatalf("want ErrRoundClosed, got %v", err)
	}
	if err := r.SetConclusion("x", RecApprove, "u", now); !errors.Is(err, ErrRoundClosed) {
		t.Fatalf("want ErrRoundClosed, got %v", err)
	}
	// double close fails
	if err := r.Close(now); !errors.Is(err, ErrRoundAlreadyClosed) {
		t.Fatalf("want ErrRoundAlreadyClosed, got %v", err)
	}
}

func TestRecommendationValid(t *testing.T) {
	if !RecApprove.Valid() || !RecReject.Valid() || !RecDefer.Valid() || !RecApproveWithConditions.Valid() {
		t.Fatal("all recommendations should be valid")
	}
	if Recommendation("").Valid() {
		t.Fatal("empty should be invalid")
	}
}

func TestEmptyConclusionRejected(t *testing.T) {
	now := time.Now()
	r, _ := NewRound("r1", "p1", "b1", "t", now)
	if err := r.SetConclusion("", RecApprove, "u", now); !errors.Is(err, ErrSummaryEmpty) {
		t.Fatalf("want ErrSummaryEmpty, got %v", err)
	}
	if err := r.SetConclusion("ok", Recommendation("xxx"), "u", now); !errors.Is(err, ErrInvalidRecommendation) {
		t.Fatalf("want ErrInvalidRecommendation, got %v", err)
	}
}
