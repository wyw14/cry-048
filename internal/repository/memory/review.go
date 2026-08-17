package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/cry048/design-review-platform/internal/domain/annotation"
	"github.com/cry048/design-review-platform/internal/domain/review"
)

// ReviewRepo is an in-memory review repository with optimistic locking.
type ReviewRepo struct {
	mu        sync.Mutex
	rounds    map[string]*review.Round
	snapshots map[string][]review.Snapshot
}

func NewReviewRepo() *ReviewRepo {
	return &ReviewRepo{
		rounds:    make(map[string]*review.Round),
		snapshots: make(map[string][]review.Snapshot),
	}
}

func (r *ReviewRepo) SaveRound(ctx context.Context, rd *review.Round) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *rd
	r.rounds[rd.ID] = &cp
	return nil
}

func (r *ReviewRepo) GetRound(ctx context.Context, id string) (*review.Round, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rd, ok := r.rounds[id]
	if !ok {
		return nil, review.ErrRoundNotFound
	}
	cp := *rd
	return &cp, nil
}

func (r *ReviewRepo) ListRounds(ctx context.Context, boardID string) ([]*review.Round, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*review.Round, 0)
	for _, rd := range r.rounds {
		if rd.BoardID == boardID {
			cp := *rd
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (r *ReviewRepo) UpdateRound(ctx context.Context, rd *review.Round, expectedVersion int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.rounds[rd.ID]
	if !ok {
		return review.ErrRoundNotFound
	}
	if existing.Version != expectedVersion {
		return fmt.Errorf("%w: round version mismatch (existing=%d expected=%d)", annotation.ErrStaleVersion, existing.Version, expectedVersion)
	}
	cp := *rd
	r.rounds[rd.ID] = &cp
	return nil
}

func (r *ReviewRepo) SaveSnapshot(ctx context.Context, s review.Snapshot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snapshots[s.RoundID] = append(r.snapshots[s.RoundID], s)
	return nil
}

func (r *ReviewRepo) ListSnapshots(ctx context.Context, roundID string) ([]review.Snapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]review.Snapshot, len(r.snapshots[roundID]))
	copy(out, r.snapshots[roundID])
	return out, nil
}
