package domain

import (
	"fmt"
	"strings"
	"time"
)

type Project struct {
	ID        ID
	Name      string
	OwnerID   ID
	CreatedAt time.Time
	Revision  Revision
}

func NewProject(id ID, name string, owner Actor, now time.Time) (Project, error) {
	name = strings.TrimSpace(name)
	if !id.Valid() || !owner.Valid() || name == "" {
		return Project{}, NewValidationError("invalid project", FieldError{Field: "name", Message: "required"})
	}
	return Project{ID: id, Name: name, OwnerID: owner.ID, CreatedAt: now, Revision: 1}, nil
}

type Board struct {
	ID        ID
	ProjectID ID
	Name      string
	Revision  Revision
}

func NewBoard(id, projectID ID, name string) (Board, error) {
	name = strings.TrimSpace(name)
	if !id.Valid() || !projectID.Valid() || name == "" {
		return Board{}, fmt.Errorf("%w: board identity and name are required", ErrInvalidArgument)
	}
	return Board{ID: id, ProjectID: projectID, Name: name, Revision: 1}, nil
}

type VersionStatus string

const (
	VersionDraft      VersionStatus = "draft"
	VersionPublished  VersionStatus = "published"
	VersionSuperseded VersionStatus = "superseded"
)

type PreviewConfig struct {
	Background string
	Scale      float64
	ShowGrid   bool
	Hotspots   []string
}

func (config PreviewConfig) Clone() PreviewConfig {
	clone := config
	clone.Hotspots = append([]string(nil), config.Hotspots...)
	return clone
}

type DesignVersion struct {
	ID           ID
	BoardID      ID
	Sequence     int
	Status       VersionStatus
	Preview      PreviewConfig
	PublishedAt  *time.Time
	SupersededBy ID
	Revision     Revision
}

func NewDesignVersion(id, boardID ID, sequence int, preview PreviewConfig) (DesignVersion, error) {
	if !id.Valid() || !boardID.Valid() || sequence < 1 {
		return DesignVersion{}, fmt.Errorf("%w: invalid design version", ErrInvalidArgument)
	}
	if preview.Scale <= 0 {
		preview.Scale = 1
	}
	return DesignVersion{ID: id, BoardID: boardID, Sequence: sequence, Status: VersionDraft, Preview: preview.Clone(), Revision: 1}, nil
}

func (version DesignVersion) Publish(now time.Time) (DesignVersion, error) {
	if version.Status != VersionDraft {
		return DesignVersion{}, fmt.Errorf("%w: only drafts can be published", ErrInvalidState)
	}
	result := version
	result.Status = VersionPublished
	result.PublishedAt = &now
	result.Revision = version.Revision.Next()
	result.Preview = version.Preview.Clone()
	return result, nil
}

func (version DesignVersion) Supersede(newVersion ID) (DesignVersion, error) {
	if version.Status != VersionPublished || !newVersion.Valid() {
		return DesignVersion{}, fmt.Errorf("%w: published version required", ErrInvalidState)
	}
	result := version
	result.Status = VersionSuperseded
	result.SupersededBy = newVersion
	result.Revision = version.Revision.Next()
	result.Preview = version.Preview.Clone()
	return result, nil
}
