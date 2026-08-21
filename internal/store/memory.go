package store

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"designreview/internal/domain"
)

type MemoryRepository struct {
	mu            sync.RWMutex
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

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		projects: make(map[domain.ID]domain.Project), boards: make(map[domain.ID]domain.Board),
		versions: make(map[domain.ID]domain.DesignVersion), annotations: make(map[domain.ID]domain.Annotation),
		reviews: make(map[domain.ID]domain.ReviewRound), members: make(map[string]domain.Member),
		attachments: make(map[domain.ID]domain.Attachment), notifications: make(map[domain.ID]domain.Notification),
	}
}

func checkContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("%w: %v", domain.ErrCanceled, ctx.Err())
	default:
		return nil
	}
}

func memberKey(projectID, actorID domain.ID) string {
	return projectID.String() + ":" + actorID.String()
}

func (repository *MemoryRepository) GetProject(ctx context.Context, id domain.ID) (domain.Project, error) {
	if err := checkContext(ctx); err != nil {
		return domain.Project{}, err
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	value, ok := repository.projects[id]
	if !ok {
		return domain.Project{}, domain.ErrNotFound
	}
	return value, nil
}

func (repository *MemoryRepository) PutProject(ctx context.Context, value domain.Project) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.projects[value.ID] = value
	return nil
}

func (repository *MemoryRepository) GetBoard(ctx context.Context, id domain.ID) (domain.Board, error) {
	if err := checkContext(ctx); err != nil {
		return domain.Board{}, err
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	value, ok := repository.boards[id]
	if !ok {
		return domain.Board{}, domain.ErrNotFound
	}
	return value, nil
}

func (repository *MemoryRepository) PutBoard(ctx context.Context, value domain.Board) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.boards[value.ID] = value
	return nil
}

func (repository *MemoryRepository) GetVersion(ctx context.Context, id domain.ID) (domain.DesignVersion, error) {
	if err := checkContext(ctx); err != nil {
		return domain.DesignVersion{}, err
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	value, ok := repository.versions[id]
	if !ok {
		return domain.DesignVersion{}, domain.ErrNotFound
	}
	value.Preview = value.Preview.Clone()
	return value, nil
}

func (repository *MemoryRepository) ListVersionsByBoard(ctx context.Context, boardID domain.ID) ([]domain.DesignVersion, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	result := make([]domain.DesignVersion, 0)
	for _, value := range repository.versions {
		if value.BoardID == boardID {
			value.Preview = value.Preview.Clone()
			result = append(result, value)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Sequence < result[j].Sequence })
	return result, nil
}

func (repository *MemoryRepository) PutVersion(ctx context.Context, value domain.DesignVersion) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	value.Preview = value.Preview.Clone()
	repository.versions[value.ID] = value
	return nil
}

func (repository *MemoryRepository) GetAnnotation(ctx context.Context, id domain.ID) (domain.Annotation, error) {
	if err := checkContext(ctx); err != nil {
		return domain.Annotation{}, err
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	value, ok := repository.annotations[id]
	if !ok {
		return domain.Annotation{}, domain.ErrNotFound
	}
	return value.Clone(), nil
}

func (repository *MemoryRepository) ListAnnotationsByVersion(ctx context.Context, versionID domain.ID) ([]domain.Annotation, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	result := make([]domain.Annotation, 0)
	for _, value := range repository.annotations {
		if value.Anchor.VersionID == versionID {
			result = append(result, value.Clone())
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID.String() < result[j].ID.String() })
	return result, nil
}

func (repository *MemoryRepository) CompareAndSwapAnnotation(ctx context.Context, value domain.Annotation, expected domain.Revision) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	current, ok := repository.annotations[value.ID]
	if !ok {
		return domain.ErrNotFound
	}
	if current.Revision != expected {
		return domain.ErrConflict
	}
	repository.annotations[value.ID] = value.Clone()
	return nil
}

func (repository *MemoryRepository) PutAnnotation(ctx context.Context, value domain.Annotation) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.annotations[value.ID] = value.Clone()
	return nil
}

func (repository *MemoryRepository) GetReviewRound(ctx context.Context, id domain.ID) (domain.ReviewRound, error) {
	if err := checkContext(ctx); err != nil {
		return domain.ReviewRound{}, err
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	value, ok := repository.reviews[id]
	if !ok {
		return domain.ReviewRound{}, domain.ErrNotFound
	}
	return value.Clone(), nil
}

func (repository *MemoryRepository) PutReviewRound(ctx context.Context, value domain.ReviewRound) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.reviews[value.ID] = value.Clone()
	return nil
}

func (repository *MemoryRepository) GetMember(ctx context.Context, projectID, actorID domain.ID) (domain.Member, error) {
	if err := checkContext(ctx); err != nil {
		return domain.Member{}, err
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	value, ok := repository.members[memberKey(projectID, actorID)]
	if !ok {
		return domain.Member{}, domain.ErrNotFound
	}
	return value, nil
}

func (repository *MemoryRepository) FindMemberByEmail(ctx context.Context, projectID domain.ID, email string) (domain.Member, error) {
	if err := checkContext(ctx); err != nil {
		return domain.Member{}, err
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	normalized := domain.NormalizeEmail(email)
	for _, value := range repository.members {
		if value.ProjectID == projectID && value.Email == normalized {
			return value, nil
		}
	}
	return domain.Member{}, domain.ErrNotFound
}

func (repository *MemoryRepository) CompareAndSwapMember(ctx context.Context, value domain.Member, expected domain.Revision) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	key := memberKey(value.ProjectID, value.ActorID)
	current, ok := repository.members[key]
	if !ok {
		return domain.ErrNotFound
	}
	if current.Revision != expected {
		return domain.ErrConflict
	}
	for otherKey, other := range repository.members {
		if otherKey != key && other.ProjectID == value.ProjectID && other.Email == value.Email {
			return domain.ErrAlreadyExists
		}
	}
	repository.members[key] = value
	return nil
}

func (repository *MemoryRepository) PutMember(ctx context.Context, value domain.Member) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	key := memberKey(value.ProjectID, value.ActorID)
	for otherKey, other := range repository.members {
		if otherKey != key && other.ProjectID == value.ProjectID && other.Email == value.Email {
			return domain.ErrAlreadyExists
		}
	}
	repository.members[key] = value
	return nil
}

func (repository *MemoryRepository) PutAttachment(ctx context.Context, value domain.Attachment) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.attachments[value.ID] = value
	return nil
}

func (repository *MemoryRepository) AppendAudit(ctx context.Context, value domain.AuditEvent) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.audits = append(repository.audits, value.Clone())
	return nil
}

func (repository *MemoryRepository) PutNotificationIfAbsent(ctx context.Context, value domain.Notification) (bool, error) {
	if err := checkContext(ctx); err != nil {
		return false, err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	for _, existing := range repository.notifications {
		if domain.NotificationKey(existing.EventID, existing.RecipientID) == domain.NotificationKey(value.EventID, value.RecipientID) {
			return false, nil
		}
	}
	repository.notifications[value.ID] = value.Clone()
	return true, nil
}

func (repository *MemoryRepository) LeaseNotifications(ctx context.Context, owner string, limit int, until time.Time) ([]domain.Notification, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	ids := make([]domain.ID, 0, len(repository.notifications))
	for id := range repository.notifications {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	result := make([]domain.Notification, 0, limit)
	now := time.Now().UTC()
	for _, id := range ids {
		value := repository.notifications[id]
		if value.SentAt != nil || (!value.LeaseExpiresAt.IsZero() && value.LeaseExpiresAt.After(now)) {
			continue
		}
		value.LeaseOwner = owner
		value.LeaseExpiresAt = until
		value.Attempt++
		repository.notifications[id] = value
		result = append(result, value.Clone())
		if len(result) == limit {
			break
		}
	}
	return result, nil
}

func (repository *MemoryRepository) MarkNotificationSent(ctx context.Context, id domain.ID, owner string, sentAt time.Time) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	value, ok := repository.notifications[id]
	if !ok {
		return domain.ErrNotFound
	}
	if value.LeaseOwner != owner {
		return domain.ErrConflict
	}
	value.SentAt = &sentAt
	value.LeaseOwner = ""
	value.LeaseExpiresAt = time.Time{}
	repository.notifications[id] = value
	return nil
}
