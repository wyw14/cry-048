// Package project defines project, board, version, and membership entities.
package project

import (
	"errors"
	"time"

	"github.com/cry048/design-review-platform/internal/domain/audit"
)

var (
	ErrProjectNotFound      = errors.New("project not found")
	ErrBoardNotFound        = errors.New("board not found")
	ErrVersionNotFound      = errors.New("version not found")
	ErrMemberNotFound       = errors.New("member not found")
	ErrMemberAlreadyExists  = errors.New("member already exists")
	ErrInvalidRole          = errors.New("invalid member role")
	ErrInvalidBoardSize     = errors.New("invalid board size")
	ErrVersionNumberInvalid = errors.New("version number must be positive")
	ErrVersionAlreadyExists = errors.New("version already exists")
	ErrBoardClosed          = errors.New("board is closed")
	ErrProjectArchived      = errors.New("project is archived")
)

// Role is a project-level member role.
type Role string

const (
	RoleOwner  Role = "owner"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

func (r Role) Valid() bool {
	switch r {
	case RoleOwner, RoleEditor, RoleViewer:
		return true
	}
	return false
}

// CanManageMembers returns true if role can manage project membership.
func (r Role) CanManageMembers() bool { return r == RoleOwner }

// CanEdit returns true if role can mutate project content.
func (r Role) CanEdit() bool { return r == RoleOwner || r == RoleEditor }

// CanAnnotate returns true if role can create or modify annotations.
func (r Role) CanAnnotate() bool { return r == RoleOwner || r == RoleEditor }

type ID = string

// Project is a top-level design review project.
type Project struct {
	ID          ID
	Name        string
	Description string
	Status      string // active | archived
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Version     int
}

func NewProject(id, name, description string, now time.Time) (*Project, error) {
	if id == "" {
		return nil, errors.New("project id required")
	}
	if name == "" {
		return nil, errors.New("project name required")
	}
	return &Project{
		ID:          id,
		Name:        name,
		Description: description,
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
		Version:     1,
	}, nil
}

func (p *Project) Archive(now time.Time) error {
	if p.Status == "archived" {
		return ErrProjectArchived
	}
	p.Status = "archived"
	p.UpdatedAt = now
	p.Version++
	return nil
}

// BoardSize is canvas dimensions in pixels.
type BoardSize struct {
	Width  int
	Height int
}

func (s BoardSize) Validate() error {
	if s.Width <= 0 || s.Height <= 0 {
		return ErrInvalidBoardSize
	}
	if s.Width > 100000 || s.Height > 100000 {
		return ErrInvalidBoardSize
	}
	return nil
}

// Board is a single canvas inside a project.
type Board struct {
	ID        ID
	ProjectID ID
	Name      string
	Size      BoardSize
	Status    string // open | closed
	CreatedAt time.Time
	UpdatedAt time.Time
	Version   int
}

func NewBoard(id, projectID, name string, size BoardSize, now time.Time) (*Board, error) {
	if id == "" || projectID == "" {
		return nil, errors.New("board id and project id required")
	}
	if name == "" {
		return nil, errors.New("board name required")
	}
	if err := size.Validate(); err != nil {
		return nil, err
	}
	return &Board{
		ID:        id,
		ProjectID: projectID,
		Name:      name,
		Size:      size,
		Status:    "open",
		CreatedAt: now,
		UpdatedAt: now,
		Version:   1,
	}, nil
}

func (b *Board) Close(now time.Time) error {
	if b.Status == "closed" {
		return ErrBoardClosed
	}
	b.Status = "closed"
	b.UpdatedAt = now
	b.Version++
	return nil
}

func (b *Board) IsOpen() bool { return b.Status == "open" }

// Version is a snapshot of a design board with its preview image.
type Version struct {
	ID         ID
	BoardID    ID
	Number     int
	Label      string
	PreviewKey string
	Notes      string
	Status     string // draft | published | superseded
	CreatedAt  time.Time
	CreatedBy  string
	Version    int
}

func NewVersion(id, boardID string, number int, label, previewKey, notes, createdBy string, now time.Time) (*Version, error) {
	if id == "" || boardID == "" {
		return nil, errors.New("version id and board id required")
	}
	if number <= 0 {
		return nil, ErrVersionNumberInvalid
	}
	if previewKey == "" {
		return nil, errors.New("preview key required")
	}
	return &Version{
		ID:         id,
		BoardID:    boardID,
		Number:     number,
		Label:      label,
		PreviewKey: previewKey,
		Notes:      notes,
		Status:     "draft",
		CreatedAt:  now,
		CreatedBy:  createdBy,
		Version:    1,
	}, nil
}

func (v *Version) Publish(now time.Time) {
	v.Status = "published"
	v.Version++
}

func (v *Version) Supersede(now time.Time) {
	v.Status = "superseded"
	v.Version++
}

func (v *Version) IsPublished() bool  { return v.Status == "published" }
func (v *Version) IsSuperseded() bool { return v.Status == "superseded" }

// Membership links a user to a project with a role.
type Membership struct {
	ID        string
	ProjectID ID
	UserID    string
	Role      Role
	JoinedAt  time.Time
}

func NewMembership(id, projectID, userID string, role Role, now time.Time) (*Membership, error) {
	if id == "" || projectID == "" || userID == "" {
		return nil, errors.New("membership id, project id, and user id required")
	}
	if !role.Valid() {
		return nil, ErrInvalidRole
	}
	return &Membership{ID: id, ProjectID: projectID, UserID: userID, Role: role, JoinedAt: now}, nil
}

// ChangeRecord tracks project/board/version/membership mutations for audit.
type ChangeRecord struct {
	ID         string
	ActorID    string
	Action     string
	EntityType string
	EntityID   string
	Before     string
	After      string
	At         time.Time
}

// toAuditRecord converts to audit entry. Bridge between domain and platform layer.
func (c ChangeRecord) ToAuditEntry() audit.Entry {
	return audit.Entry{
		ActorID:    c.ActorID,
		Action:     c.Action,
		EntityType: c.EntityType,
		EntityID:   c.EntityID,
		Before:     c.Before,
		After:      c.After,
		At:         c.At,
	}
}
