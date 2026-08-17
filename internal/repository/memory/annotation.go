package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cry048/design-review-platform/internal/application"
	"github.com/cry048/design-review-platform/internal/domain/annotation"
	"github.com/cry048/design-review-platform/internal/domain/canvas"
)

// AnnotationRepo is an in-memory annotation repository with optimistic locking.
type AnnotationRepo struct {
	mu          sync.Mutex
	annotations map[string]*annotation.Annotation
}

func NewAnnotationRepo() *AnnotationRepo {
	return &AnnotationRepo{annotations: make(map[string]*annotation.Annotation)}
}

func (r *AnnotationRepo) Save(ctx context.Context, a *annotation.Annotation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *a
	r.annotations[a.ID] = &cp
	return nil
}

func (r *AnnotationRepo) Get(ctx context.Context, id string) (*annotation.Annotation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.annotations[id]
	if !ok {
		return nil, annotation.ErrAnnotationNotFound
	}
	cp := *a
	return &cp, nil
}

func (r *AnnotationRepo) Update(ctx context.Context, a *annotation.Annotation, expectedVersion int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.annotations[a.ID]
	if !ok {
		return annotation.ErrAnnotationNotFound
	}
	if existing.Version != expectedVersion {
		return fmt.Errorf("%w: existing=%d expected=%d", annotation.ErrStaleVersion, existing.Version, expectedVersion)
	}
	cp := *a
	// Persist the new version (which the domain layer has already bumped on a.Version).
	r.annotations[a.ID] = &cp
	return nil
}

func (r *AnnotationRepo) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.annotations[id]; !ok {
		return annotation.ErrAnnotationNotFound
	}
	delete(r.annotations, id)
	return nil
}

func (r *AnnotationRepo) List(ctx context.Context, filter application.AnnotationFilter) ([]*annotation.Annotation, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*annotation.Annotation, 0)
	for _, a := range r.annotations {
		if filter.ProjectID != "" && a.ProjectID != filter.ProjectID {
			continue
		}
		if filter.BoardID != "" && a.BoardID != filter.BoardID {
			continue
		}
		if filter.VersionID != "" && a.VersionID != filter.VersionID {
			continue
		}
		if filter.Status != "" && string(a.Status) != filter.Status {
			continue
		}
		if filter.Priority != "" && string(a.Priority) != filter.Priority {
			continue
		}
		if filter.AssigneeID != "" && a.AssigneeID != filter.AssigneeID {
			continue
		}
		if filter.ReporterID != "" && a.ReporterID != filter.ReporterID {
			continue
		}
		if !filter.From.IsZero() && a.CreatedAt.Before(filter.From) {
			continue
		}
		if !filter.To.IsZero() && a.CreatedAt.After(filter.To) {
			continue
		}
		cp := *a
		out = append(out, &cp)
	}
	total := len(out)
	sortBy := filter.SortBy
	if sortBy == "" || !annotation.SortWhitelist[sortBy] {
		sortBy = "created_at"
	}
	sort.SliceStable(out, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "created_at":
			less = out[i].CreatedAt.Before(out[j].CreatedAt)
		case "updated_at":
			less = out[i].UpdatedAt.Before(out[j].UpdatedAt)
		case "priority":
			less = priorityRank(string(out[i].Priority)) < priorityRank(string(out[j].Priority))
		case "status":
			less = string(out[i].Status) < string(out[j].Status)
		case "due_at":
			ai, bi := out[i].DueAt, out[j].DueAt
			if ai == nil && bi == nil {
				less = out[i].CreatedAt.Before(out[j].CreatedAt)
			} else if ai == nil {
				less = false
			} else if bi == nil {
				less = true
			} else {
				less = ai.Before(*bi)
			}
		default:
			less = out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		if filter.SortDesc {
			return !less
		}
		return less
	})
	page := filter.Page
	if page < 1 {
		page = 1
	}
	size := filter.PageSize
	if size <= 0 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	start := (page - 1) * size
	if start >= len(out) {
		return []*annotation.Annotation{}, total, nil
	}
	end := start + size
	if end > len(out) {
		end = len(out)
	}
	return out[start:end], total, nil
}

func (r *AnnotationRepo) Search(ctx context.Context, q string, limit int) ([]*annotation.Annotation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	out := make([]*annotation.Annotation, 0)
	needle := strings.ToLower(q)
	for _, a := range r.annotations {
		hay := strings.ToLower(a.Title + " " + a.Body)
		if strings.Contains(hay, needle) {
			cp := *a
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *AnnotationRepo) ListByAssignee(ctx context.Context, userID string) ([]*annotation.Annotation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*annotation.Annotation, 0)
	for _, a := range r.annotations {
		if a.AssigneeID == userID && a.Status != annotation.StatusClosed && a.Status != annotation.StatusResolved {
			cp := *a
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.Before(out[j].UpdatedAt) })
	return out, nil
}

func (r *AnnotationRepo) BulkMigrateAnchors(ctx context.Context, fromVersionID, toVersionID, by, reason string, now time.Time) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, a := range r.annotations {
		if a.VersionID != fromVersionID {
			continue
		}
		var newAnchor canvas.Anchor
		if a.Anchor.Point != nil {
			newAnchor = canvas.NewPointAnchor(toVersionID, *a.Anchor.Point)
		} else if a.Anchor.Region != nil {
			newAnchor = canvas.NewRegionAnchor(toVersionID, *a.Anchor.Region)
		} else {
			newAnchor = canvas.Anchor{VersionID: toVersionID}
		}
		if _, err := a.MigrateAnchor(toVersionID, newAnchor, reason, by, now); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func priorityRank(p string) int {
	switch p {
	case "critical":
		return 0
	case "high":
		return 1
	case "normal":
		return 2
	case "low":
		return 3
	}
	return 4
}
