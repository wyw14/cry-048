// Package memory provides an in-memory repository implementation for tests and offline runs.
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
	"github.com/cry048/design-review-platform/internal/domain/project"
)

// === Project repository ===

type ProjectRepo struct {
	mu          sync.Mutex
	projects    map[string]*project.Project
	boards      map[string]*project.Board
	versions    map[string]*project.Version
	memberships map[string]*project.Membership
}

func NewProjectRepo() *ProjectRepo {
	return &ProjectRepo{
		projects:    make(map[string]*project.Project),
		boards:      make(map[string]*project.Board),
		versions:    make(map[string]*project.Version),
		memberships: make(map[string]*project.Membership),
	}
}

func (r *ProjectRepo) SaveProject(ctx context.Context, p *project.Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.projects[p.ID]; ok && existing.Version != p.Version-1 && p.Version != 1 {
		return fmt.Errorf("%w: project version mismatch", annotation.ErrStaleVersion)
	}
	cp := *p
	r.projects[p.ID] = &cp
	return nil
}

func (r *ProjectRepo) GetProject(ctx context.Context, id string) (*project.Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.projects[id]
	if !ok {
		return nil, project.ErrProjectNotFound
	}
	cp := *p
	return &cp, nil
}

func (r *ProjectRepo) ListProjects(ctx context.Context, filter application.ProjectFilter) ([]*project.Project, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*project.Project, 0)
	for _, p := range r.projects {
		if filter.Status != "" && p.Status != filter.Status {
			continue
		}
		if filter.Name != "" && !strings.Contains(p.Name, filter.Name) {
			continue
		}
		cp := *p
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, len(out), nil
}

func (r *ProjectRepo) ArchiveProject(ctx context.Context, id string, version int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.projects[id]
	if !ok {
		return project.ErrProjectNotFound
	}
	if p.Version != version {
		return fmt.Errorf("%w: project version mismatch", annotation.ErrStaleVersion)
	}
	p.Status = "archived"
	p.Version++
	p.UpdatedAt = time.Now()
	return nil
}

func (r *ProjectRepo) SaveBoard(ctx context.Context, b *project.Board) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.projects[b.ProjectID]; !ok {
		return project.ErrProjectNotFound
	}
	cp := *b
	r.boards[b.ID] = &cp
	return nil
}

func (r *ProjectRepo) GetBoard(ctx context.Context, id string) (*project.Board, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.boards[id]
	if !ok {
		return nil, project.ErrBoardNotFound
	}
	cp := *b
	return &cp, nil
}

func (r *ProjectRepo) ListBoards(ctx context.Context, projectID string) ([]*project.Board, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*project.Board, 0)
	for _, b := range r.boards {
		if b.ProjectID == projectID {
			cp := *b
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (r *ProjectRepo) UpdateBoard(ctx context.Context, b *project.Board) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.boards[b.ID]
	if !ok {
		return project.ErrBoardNotFound
	}
	if existing.Version != b.Version-1 {
		return fmt.Errorf("%w: board version mismatch", annotation.ErrStaleVersion)
	}
	cp := *b
	r.boards[b.ID] = &cp
	return nil
}

func (r *ProjectRepo) CloseBoard(ctx context.Context, id string, version int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.boards[id]
	if !ok {
		return project.ErrBoardNotFound
	}
	if b.Version != version {
		return fmt.Errorf("%w: board version mismatch", annotation.ErrStaleVersion)
	}
	b.Status = "closed"
	b.Version++
	b.UpdatedAt = time.Now()
	return nil
}

func (r *ProjectRepo) SaveVersion(ctx context.Context, v *project.Version) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.boards[v.BoardID]; !ok {
		return project.ErrBoardNotFound
	}
	for _, existing := range r.versions {
		if existing.BoardID == v.BoardID && existing.Number == v.Number {
			return project.ErrVersionAlreadyExists
		}
	}
	cp := *v
	r.versions[v.ID] = &cp
	return nil
}

func (r *ProjectRepo) GetVersion(ctx context.Context, id string) (*project.Version, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.versions[id]
	if !ok {
		return nil, project.ErrVersionNotFound
	}
	cp := *v
	return &cp, nil
}

func (r *ProjectRepo) ListVersions(ctx context.Context, boardID string) ([]*project.Version, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*project.Version, 0)
	for _, v := range r.versions {
		if v.BoardID == boardID {
			cp := *v
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return out, nil
}

// LatestVersion returns the version with the highest published number for a board.
// Used by PublishVersion to determine which prior published version to supersede.
func (r *ProjectRepo) LatestVersion(ctx context.Context, boardID string) (*project.Version, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var best *project.Version
	for _, v := range r.versions {
		if v.BoardID != boardID {
			continue
		}
		if best == nil || v.Number > best.Number {
			cp := *v
			best = &cp
		}
	}
	if best == nil {
		return nil, project.ErrVersionNotFound
	}
	return best, nil
}

func (r *ProjectRepo) UpdateVersion(ctx context.Context, v *project.Version) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.versions[v.ID]
	if !ok {
		return project.ErrVersionNotFound
	}
	if existing.Version != v.Version-1 {
		return fmt.Errorf("%w: version mismatch", annotation.ErrStaleVersion)
	}
	cp := *v
	r.versions[v.ID] = &cp
	// re-sort by number so LatestVersion returns the highest
	return nil
}

func (r *ProjectRepo) SaveMembership(ctx context.Context, m *project.Membership) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.projects[m.ProjectID]; !ok {
		return project.ErrProjectNotFound
	}
	for _, existing := range r.memberships {
		if existing.ProjectID == m.ProjectID && existing.UserID == m.UserID {
			return project.ErrMemberAlreadyExists
		}
	}
	cp := *m
	r.memberships[m.ID] = &cp
	return nil
}

func (r *ProjectRepo) GetMembership(ctx context.Context, projectID, userID string) (*project.Membership, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, m := range r.memberships {
		if m.ProjectID == projectID && m.UserID == userID {
			cp := *m
			return &cp, nil
		}
	}
	return nil, project.ErrMemberNotFound
}

func (r *ProjectRepo) ListMemberships(ctx context.Context, projectID string) ([]*project.Membership, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*project.Membership, 0)
	for _, m := range r.memberships {
		if m.ProjectID == projectID {
			cp := *m
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].JoinedAt.Before(out[j].JoinedAt) })
	return out, nil
}

func (r *ProjectRepo) DeleteMembership(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.memberships[id]; !ok {
		return project.ErrMemberNotFound
	}
	delete(r.memberships, id)
	return nil
}
