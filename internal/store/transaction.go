package store

import (
	"context"
	"sync"

	"designreview/internal/domain"
)

type MemoryUnitOfWork struct {
	repository *MemoryRepository
	mu         sync.Mutex
}

func NewMemoryUnitOfWork(repository *MemoryRepository) *MemoryUnitOfWork {
	return &MemoryUnitOfWork{repository: repository}
}

func (unit *MemoryUnitOfWork) Within(ctx context.Context, operation func(context.Context, Repository) error) error {
	return unit.withLegacyCommit(ctx, operation)
}

func (unit *MemoryUnitOfWork) withinAtomic(ctx context.Context, operation func(context.Context, Repository) error) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	unit.mu.Lock()
	defer unit.mu.Unlock()
	snapshot := unit.repository.snapshot()
	if err := operation(ctx, unit.repository); err != nil {
		unit.repository.restore(snapshot)
		return err
	}
	if err := checkContext(ctx); err != nil {
		unit.repository.restore(snapshot)
		return err
	}
	return nil
}

type memorySnapshot struct {
	projects      map[domain.ID]domain.Project
	boards        map[domain.ID]domain.Board
	versions      map[domain.ID]domain.DesignVersion
	annotations   map[domain.ID]domain.Annotation
	reviews       map[domain.ID]domain.ReviewRound
	members       map[string]domain.Member
	attachments   map[domain.ID]domain.Attachment
	audits        []domain.AuditEvent
	notifications map[domain.ID]domain.Notification
}

func (repository *MemoryRepository) snapshot() memorySnapshot {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	result := memorySnapshot{
		projects: make(map[domain.ID]domain.Project, len(repository.projects)), boards: make(map[domain.ID]domain.Board, len(repository.boards)),
		versions: make(map[domain.ID]domain.DesignVersion, len(repository.versions)), annotations: make(map[domain.ID]domain.Annotation, len(repository.annotations)),
		reviews: make(map[domain.ID]domain.ReviewRound, len(repository.reviews)), members: make(map[string]domain.Member, len(repository.members)),
		attachments: make(map[domain.ID]domain.Attachment, len(repository.attachments)), audits: make([]domain.AuditEvent, len(repository.audits)),
		notifications: make(map[domain.ID]domain.Notification, len(repository.notifications)),
	}
	for key, value := range repository.projects {
		result.projects[key] = value
	}
	for key, value := range repository.boards {
		result.boards[key] = value
	}
	for key, value := range repository.versions {
		value.Preview = value.Preview.Clone()
		result.versions[key] = value
	}
	for key, value := range repository.annotations {
		result.annotations[key] = value.Clone()
	}
	for key, value := range repository.reviews {
		result.reviews[key] = value.Clone()
	}
	for key, value := range repository.members {
		result.members[key] = value
	}
	for key, value := range repository.attachments {
		result.attachments[key] = value
	}
	for index, value := range repository.audits {
		result.audits[index] = value.Clone()
	}
	for key, value := range repository.notifications {
		result.notifications[key] = value.Clone()
	}
	return result
}

func (repository *MemoryRepository) restore(snapshot memorySnapshot) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.projects = snapshot.projects
	repository.boards = snapshot.boards
	repository.versions = snapshot.versions
	repository.annotations = snapshot.annotations
	repository.reviews = snapshot.reviews
	repository.members = snapshot.members
	repository.attachments = snapshot.attachments
	repository.audits = snapshot.audits
	repository.notifications = snapshot.notifications
}
