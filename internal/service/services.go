// Package service wires use cases to repositories, audit, notifier, and storage.
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cry048/design-review-platform/internal/application"
	"github.com/cry048/design-review-platform/internal/domain/annotation"
	"github.com/cry048/design-review-platform/internal/domain/audit"
	"github.com/cry048/design-review-platform/internal/domain/canvas"
	"github.com/cry048/design-review-platform/internal/domain/project"
	"github.com/cry048/design-review-platform/internal/domain/review"
	"github.com/cry048/design-review-platform/internal/platform/notify"
	"github.com/cry048/design-review-platform/internal/platform/storage"
)

// ProjectService implements project/board/version/member use cases.
type ProjectService struct {
	Projects application.ProjectRepository
	Audit    application.AuditLogger
	Clock    application.Clock
	IDs      application.IDGenerator
}

func (s *ProjectService) CreateProject(ctx context.Context, in application.CreateProjectInput) (*project.Project, error) {
	now := s.Clock.Now()
	id := in.ID
	if id == "" {
		id = s.IDs.New()
	}
	p, err := project.NewProject(id, in.Name, in.Description, now)
	if err != nil {
		return nil, err
	}
	if err := s.Projects.SaveProject(ctx, p); err != nil {
		return nil, err
	}
	_ = s.Audit.Log(ctx, "project.create", "system", "project", p.ID, "", p.Name)
	return p, nil
}

func (s *ProjectService) GetProject(ctx context.Context, id string) (*project.Project, error) {
	return s.Projects.GetProject(ctx, id)
}

func (s *ProjectService) ListProjects(ctx context.Context, filter application.ProjectFilter) ([]*project.Project, int, error) {
	return s.Projects.ListProjects(ctx, filter)
}

func (s *ProjectService) ArchiveProject(ctx context.Context, id, actorID string) error {
	p, err := s.Projects.GetProject(ctx, id)
	if err != nil {
		return err
	}
	before := p.Status
	if err := p.Archive(s.Clock.Now()); err != nil {
		return err
	}
	if err := s.Projects.ArchiveProject(ctx, id, p.Version-1); err != nil {
		return err
	}
	_ = s.Audit.Log(ctx, "project.archive", actorID, "project", id, before, p.Status)
	return nil
}

func (s *ProjectService) CreateBoard(ctx context.Context, in application.CreateBoardInput) (*project.Board, error) {
	if _, err := s.Projects.GetProject(ctx, in.ProjectID); err != nil {
		return nil, err
	}
	now := s.Clock.Now()
	id := in.ID
	if id == "" {
		id = s.IDs.New()
	}
	b, err := project.NewBoard(id, in.ProjectID, in.Name, project.BoardSize{Width: in.Width, Height: in.Height}, now)
	if err != nil {
		return nil, err
	}
	if err := s.Projects.SaveBoard(ctx, b); err != nil {
		return nil, err
	}
	_ = s.Audit.Log(ctx, "board.create", "system", "board", b.ID, "", b.Name)
	return b, nil
}

func (s *ProjectService) GetBoard(ctx context.Context, id string) (*project.Board, error) {
	return s.Projects.GetBoard(ctx, id)
}

func (s *ProjectService) ListBoards(ctx context.Context, projectID string) ([]*project.Board, error) {
	return s.Projects.ListBoards(ctx, projectID)
}

func (s *ProjectService) CloseBoard(ctx context.Context, id, actorID string) error {
	b, err := s.Projects.GetBoard(ctx, id)
	if err != nil {
		return err
	}
	before := b.Status
	if err := b.Close(s.Clock.Now()); err != nil {
		return err
	}
	if err := s.Projects.CloseBoard(ctx, id, b.Version-1); err != nil {
		return err
	}
	_ = s.Audit.Log(ctx, "board.close", actorID, "board", id, before, b.Status)
	return nil
}

func (s *ProjectService) CreateVersion(ctx context.Context, in application.CreateVersionInput) (*project.Version, error) {
	b, err := s.Projects.GetBoard(ctx, in.BoardID)
	if err != nil {
		return nil, err
	}
	if !b.IsOpen() {
		return nil, project.ErrBoardClosed
	}
	now := s.Clock.Now()
	id := in.ID
	if id == "" {
		id = s.IDs.New()
	}
	v, err := project.NewVersion(id, in.BoardID, in.Number, in.Label, in.PreviewKey, in.Notes, in.CreatedBy, now)
	if err != nil {
		return nil, err
	}
	if err := s.Projects.SaveVersion(ctx, v); err != nil {
		return nil, err
	}
	_ = s.Audit.Log(ctx, "version.create", in.CreatedBy, "version", v.ID, "", fmt.Sprintf("v%d", v.Number))
	return v, nil
}

func (s *ProjectService) GetVersion(ctx context.Context, id string) (*project.Version, error) {
	return s.Projects.GetVersion(ctx, id)
}

func (s *ProjectService) ListVersions(ctx context.Context, boardID string) ([]*project.Version, error) {
	return s.Projects.ListVersions(ctx, boardID)
}

// PublishVersion sets the version to published; superseding any currently-published version on the same board.
// The caller is responsible for migrating anchors separately.
func (s *ProjectService) PublishVersion(ctx context.Context, id, actorID string) (*project.Version, error) {
	v, err := s.Projects.GetVersion(ctx, id)
	if err != nil {
		return nil, err
	}
	now := s.Clock.Now()
	all, err := s.Projects.ListVersions(ctx, v.BoardID)
	if err != nil {
		return nil, err
	}
	for _, other := range all {
		if other.ID == v.ID {
			continue
		}
		if other.IsPublished() {
			other.Supersede(now)
			if err := s.Projects.UpdateVersion(ctx, other); err != nil {
				return nil, err
			}
		}
	}
	v.Publish(now)
	if err := s.Projects.UpdateVersion(ctx, v); err != nil {
		return nil, err
	}
	_ = s.Audit.Log(ctx, "version.publish", actorID, "version", v.ID, "draft", "published")
	return v, nil
}

func (s *ProjectService) AddMember(ctx context.Context, in application.AddMemberInput) (*project.Membership, error) {
	if _, err := s.Projects.GetProject(ctx, in.ProjectID); err != nil {
		return nil, err
	}
	role := project.Role(in.Role)
	if !role.Valid() {
		return nil, project.ErrInvalidRole
	}
	now := s.Clock.Now()
	id := in.ID
	if id == "" {
		id = s.IDs.New()
	}
	m, err := project.NewMembership(id, in.ProjectID, in.UserID, role, now)
	if err != nil {
		return nil, err
	}
	if err := s.Projects.SaveMembership(ctx, m); err != nil {
		return nil, err
	}
	_ = s.Audit.Log(ctx, "member.add", "system", "membership", m.ID, "", string(m.Role))
	return m, nil
}

func (s *ProjectService) ListMembers(ctx context.Context, projectID string) ([]*project.Membership, error) {
	return s.Projects.ListMemberships(ctx, projectID)
}

func (s *ProjectService) RemoveMember(ctx context.Context, id, actorID string) error {
	if err := s.Projects.DeleteMembership(ctx, id); err != nil {
		return err
	}
	_ = s.Audit.Log(ctx, "member.remove", actorID, "membership", id, "", "")
	return nil
}

// AnnotationService implements annotation use cases and enforces invariants.
type AnnotationService struct {
	Projects            application.ProjectRepository
	Annotations         application.AnnotationRepository
	Reviews             application.ReviewRepository
	Audit               application.AuditLogger
	Notifier            application.Notifier
	Storage             *storage.Store
	Clock               application.Clock
	IDs                 application.IDGenerator
	AttachmentValidator application.AttachmentValidator
	TxRunner            application.TxRunner
}

// validateAnchor ensures the supplied anchor is well-formed for the version's board.
func (s *AnnotationService) validateAnchor(ctx context.Context, boardID string, v *project.Version, anchor canvas.Anchor) error {
	b, err := s.Projects.GetBoard(ctx, boardID)
	if err != nil {
		return err
	}
	if anchor.Region != nil {
		if err := anchor.Region.Validate(b.Size.Width, b.Size.Height); err != nil {
			return err
		}
	}
	if anchor.Point != nil {
		if err := anchor.Point.Validate(b.Size.Width, b.Size.Height); err != nil {
			return err
		}
	}
	if anchor.VersionID != v.ID {
		return errors.New("anchor version does not match requested version")
	}
	return nil
}

func (s *AnnotationService) Create(ctx context.Context, in application.CreateAnnotationInput) (*annotation.Annotation, error) {
	p, err := s.Projects.GetProject(ctx, in.ProjectID)
	if err != nil {
		return nil, err
	}
	if p.Status == "archived" {
		return nil, project.ErrProjectArchived
	}
	b, err := s.Projects.GetBoard(ctx, in.BoardID)
	if err != nil {
		return nil, err
	}
	if !b.IsOpen() {
		return nil, project.ErrBoardClosed
	}
	v, err := s.Projects.GetVersion(ctx, in.VersionID)
	if err != nil {
		return nil, err
	}
	if !v.IsPublished() {
		return nil, errors.New("cannot annotate a non-published version")
	}
	var anchor canvas.Anchor
	if in.Point != nil {
		anchor = canvas.NewPointAnchor(in.VersionID, *in.Point)
	} else if in.Region != nil {
		anchor = canvas.NewRegionAnchor(in.VersionID, *in.Region)
	} else {
		return nil, errors.New("annotation requires a point or region anchor")
	}
	if err := s.validateAnchor(ctx, in.BoardID, v, anchor); err != nil {
		return nil, err
	}
	now := s.Clock.Now()
	id := in.ID
	if id == "" {
		id = s.IDs.New()
	}
	priority := annotation.Priority(in.Priority)
	if priority == "" {
		priority = annotation.PriorityNormal
	}
	a, err := annotation.NewAnnotation(id, in.ProjectID, in.BoardID, in.VersionID, in.Title, in.Body, in.ReporterID, anchor, priority, now)
	if err != nil {
		return nil, err
	}
	if in.AssigneeID != "" {
		a.SetAssignee(in.AssigneeID, now)
	}
	if in.DueAt != nil {
		if err := a.SetDueAt(in.DueAt, now); err != nil {
			return nil, err
		}
	}
	if err := s.Annotations.Save(ctx, a); err != nil {
		return nil, err
	}
	_ = s.Audit.Log(ctx, "annotation.create", in.ReporterID, "annotation", a.ID, "", a.Title)
	if in.AssigneeID != "" {
		_ = s.Notifier.NotifyAnnotationAssigned(ctx, a)
	}
	return a, nil
}

func (s *AnnotationService) Get(ctx context.Context, id string) (*annotation.Annotation, error) {
	return s.Annotations.Get(ctx, id)
}

func (s *AnnotationService) List(ctx context.Context, filter application.AnnotationFilter) ([]*annotation.Annotation, int, error) {
	return s.Annotations.List(ctx, filter)
}

func (s *AnnotationService) Search(ctx context.Context, q string, limit int) ([]*annotation.Annotation, error) {
	return s.Annotations.Search(ctx, q, limit)
}

func (s *AnnotationService) ListForUser(ctx context.Context, userID string) ([]*annotation.Annotation, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, annotation.ErrAssigneeRequired
	}
	if s.Clock == nil {
		return nil, errors.New("annotation service clock is required")
	}
	return s.Annotations.ListTodos(ctx, application.TodoQuery{
		AssigneeID: userID,
		Now:        s.Clock.Now(),
		Limit:      100,
	})
}

func (s *AnnotationService) AddReply(ctx context.Context, in application.AddReplyInput) (*annotation.Annotation, error) {
	a, err := s.Annotations.Get(ctx, in.AnnotationID)
	if err != nil {
		return nil, err
	}
	now := s.Clock.Now()
	id := s.IDs.New()
	expected := a.Version
	if _, err := a.AddReply(id, in.AuthorID, in.Body, now); err != nil {
		return nil, err
	}
	if err := s.Annotations.Update(ctx, a, expected); err != nil {
		return nil, err
	}
	_ = s.Audit.Log(ctx, "annotation.reply", in.AuthorID, "annotation", a.ID, "", "")
	return a, nil
}

func (s *AnnotationService) SetAssignee(ctx context.Context, annotationID, assigneeID, actorID string) (*annotation.Annotation, error) {
	a, err := s.Annotations.Get(ctx, annotationID)
	if err != nil {
		return nil, err
	}
	expected := a.Version
	a.SetAssignee(assigneeID, s.Clock.Now())
	if err := s.Annotations.Update(ctx, a, expected); err != nil {
		return nil, err
	}
	_ = s.Audit.Log(ctx, "annotation.assign", actorID, "annotation", a.ID, "", assigneeID)
	if assigneeID != "" {
		_ = s.Notifier.NotifyAnnotationAssigned(ctx, a)
	}
	return a, nil
}

func (s *AnnotationService) SetDueAt(ctx context.Context, annotationID string, due *time.Time, actorID string) (*annotation.Annotation, error) {
	a, err := s.Annotations.Get(ctx, annotationID)
	if err != nil {
		return nil, err
	}
	expected := a.Version
	if err := a.SetDueAt(due, s.Clock.Now()); err != nil {
		return nil, err
	}
	if err := s.Annotations.Update(ctx, a, expected); err != nil {
		return nil, err
	}
	_ = s.Audit.Log(ctx, "annotation.due", actorID, "annotation", a.ID, "", "")
	return a, nil
}

func (s *AnnotationService) SetPriority(ctx context.Context, annotationID, priority, actorID string) (*annotation.Annotation, error) {
	a, err := s.Annotations.Get(ctx, annotationID)
	if err != nil {
		return nil, err
	}
	expected := a.Version
	if err := a.SetPriority(annotation.Priority(priority), s.Clock.Now()); err != nil {
		return nil, err
	}
	if err := s.Annotations.Update(ctx, a, expected); err != nil {
		return nil, err
	}
	_ = s.Audit.Log(ctx, "annotation.priority", actorID, "annotation", a.ID, "", priority)
	return a, nil
}

func (s *AnnotationService) RequestReview(ctx context.Context, in application.RequestReviewInput) (*annotation.Annotation, error) {
	a, err := s.Annotations.Get(ctx, in.AnnotationID)
	if err != nil {
		return nil, err
	}
	expected := a.Version
	if err := a.RequestReview(s.Clock.Now(), in.ActorID); err != nil {
		return nil, err
	}
	if err := s.Annotations.Update(ctx, a, expected); err != nil {
		return nil, err
	}
	_ = s.Audit.Log(ctx, "annotation.request_review", in.ActorID, "annotation", a.ID, "replied", "review")
	return a, nil
}

func (s *AnnotationService) Resolve(ctx context.Context, in application.ResolveAnnotationInput) (*annotation.Annotation, error) {
	a, err := s.Annotations.Get(ctx, in.AnnotationID)
	if err != nil {
		return nil, err
	}
	expected := a.Version
	if err := a.Resolve(s.Clock.Now(), in.ActorID); err != nil {
		return nil, err
	}
	if err := s.Annotations.Update(ctx, a, expected); err != nil {
		return nil, err
	}
	_ = s.Audit.Log(ctx, "annotation.resolve", in.ActorID, "annotation", a.ID, "review", "resolved")
	_ = s.Notifier.NotifyAnnotationResolved(ctx, a)
	return a, nil
}

// Reopen is the only path back from resolved status (via复核).
func (s *AnnotationService) Reopen(ctx context.Context, in application.ReopenAnnotationInput) (*annotation.Annotation, error) {
	a, err := s.Annotations.Get(ctx, in.AnnotationID)
	if err != nil {
		return nil, err
	}
	if a.Status != annotation.StatusResolved {
		return nil, annotation.ErrCannotReopenUnresolved
	}
	expected := a.Version
	if err := a.Reopen(s.Clock.Now(), in.ActorID, in.Reason); err != nil {
		return nil, err
	}
	if err := s.Annotations.Update(ctx, a, expected); err != nil {
		return nil, err
	}
	_ = s.Audit.Log(ctx, "annotation.reopen", in.ActorID, "annotation", a.ID, "resolved", "open")
	_ = s.Notifier.NotifyAnnotationReopened(ctx, a)
	return a, nil
}

func (s *AnnotationService) Close(ctx context.Context, in application.CloseAnnotationInput) (*annotation.Annotation, error) {
	a, err := s.Annotations.Get(ctx, in.AnnotationID)
	if err != nil {
		return nil, err
	}
	expected := a.Version
	if err := a.Close(s.Clock.Now(), in.ActorID); err != nil {
		return nil, err
	}
	if err := s.Annotations.Update(ctx, a, expected); err != nil {
		return nil, err
	}
	_ = s.Audit.Log(ctx, "annotation.close", in.ActorID, "annotation", a.ID, string(a.Status), "closed")
	return a, nil
}

func (s *AnnotationService) AddAttachment(ctx context.Context, in application.AddAttachmentInput) (*annotation.Annotation, error) {
	a, err := s.Annotations.Get(ctx, in.AnnotationID)
	if err != nil {
		return nil, err
	}
	if s.AttachmentValidator != nil {
		if err := s.AttachmentValidator.ValidateUpload(in.MediaType, in.Size); err != nil {
			return nil, err
		}
	}
	expected := a.Version
	a.AddAttachment(annotation.Attachment{
		ID:         s.IDs.New(),
		Filename:   in.Filename,
		MediaType:  in.MediaType,
		Size:       in.Size,
		StorageKey: in.StorageKey,
		UploadedBy: in.UploadedBy,
	}, s.Clock.Now())
	if err := s.Annotations.Update(ctx, a, expected); err != nil {
		return nil, err
	}
	_ = s.Audit.Log(ctx, "annotation.attachment_add", in.UploadedBy, "annotation", a.ID, "", in.Filename)
	return a, nil
}

// MigrateAnchors moves all anchors on fromVersionID to toVersionID atomically.
// Critical invariant: version replacement MUST NOT silently drop original annotations.
func (s *AnnotationService) MigrateAnchors(ctx context.Context, in application.MigrateAnchorsInput) (int, error) {
	if in.FromVersionID == "" || in.ToVersionID == "" {
		return 0, errors.New("from and to version ids required")
	}
	if in.FromVersionID == in.ToVersionID {
		return 0, errors.New("from and to versions must differ")
	}
	to, err := s.Projects.GetVersion(ctx, in.ToVersionID)
	if err != nil {
		return 0, err
	}
	if !to.IsPublished() {
		return 0, errors.New("target version must be published")
	}
	from, err := s.Projects.GetVersion(ctx, in.FromVersionID)
	if err != nil {
		return 0, err
	}
	if from.BoardID != to.BoardID {
		return 0, errors.New("versions belong to different boards")
	}
	if s.TxRunner != nil {
		var count int
		err = s.TxRunner.RunInTx(ctx, func(ctx context.Context) error {
			c, err := s.Annotations.BulkMigrateAnchors(ctx, in.FromVersionID, in.ToVersionID, in.By, in.Reason, s.Clock.Now())
			if err != nil {
				return err
			}
			count = c
			return nil
		})
		if err != nil {
			return 0, err
		}
		_ = s.Audit.Log(ctx, "annotation.migrate_anchors", in.By, "version", in.FromVersionID, in.FromVersionID, in.ToVersionID)
		return count, nil
	}
	count, err := s.Annotations.BulkMigrateAnchors(ctx, in.FromVersionID, in.ToVersionID, in.By, in.Reason, s.Clock.Now())
	if err != nil {
		return 0, err
	}
	_ = s.Audit.Log(ctx, "annotation.migrate_anchors", in.By, "version", in.FromVersionID, in.FromVersionID, in.ToVersionID)
	return count, nil
}

// MarkStale flags annotations on a now-superseded version as stale.
func (s *AnnotationService) MarkStale(ctx context.Context, versionID, reason string) error {
	list, _, err := s.Annotations.List(ctx, application.AnnotationFilter{VersionID: versionID, PageSize: 1000})
	if err != nil {
		return err
	}
	now := s.Clock.Now()
	for _, a := range list {
		expected := a.Version
		a.MarkStale(reason, now)
		if err := s.Annotations.Update(ctx, a, expected); err != nil {
			return err
		}
	}
	return nil
}

// ReviewService implements review rounds, snapshots, conclusions, and closure.
type ReviewService struct {
	Projects    application.ProjectRepository
	Annotations application.AnnotationRepository
	Reviews     application.ReviewRepository
	Audit       application.AuditLogger
	Clock       application.Clock
	IDs         application.IDGenerator
	TxRunner    application.TxRunner
}

func (s *ReviewService) CreateRound(ctx context.Context, in application.CreateRoundInput) (*review.Round, error) {
	if _, err := s.Projects.GetProject(ctx, in.ProjectID); err != nil {
		return nil, err
	}
	if _, err := s.Projects.GetBoard(ctx, in.BoardID); err != nil {
		return nil, err
	}
	id := in.ID
	if id == "" {
		id = s.IDs.New()
	}
	rd, err := review.NewRound(id, in.ProjectID, in.BoardID, in.Title, s.Clock.Now())
	if err != nil {
		return nil, err
	}
	if err := s.Reviews.SaveRound(ctx, rd); err != nil {
		return nil, err
	}
	_ = s.Audit.Log(ctx, "review.round_create", "system", "review_round", rd.ID, "", rd.Title)
	return rd, nil
}

func (s *ReviewService) GetRound(ctx context.Context, id string) (*review.Round, error) {
	return s.Reviews.GetRound(ctx, id)
}

func (s *ReviewService) ListRounds(ctx context.Context, boardID string) ([]*review.Round, error) {
	return s.Reviews.ListRounds(ctx, boardID)
}

// CreateSnapshot counts annotations by status and freezes them.
func (s *ReviewService) CreateSnapshot(ctx context.Context, in application.SnapshotInput) (review.Snapshot, error) {
	rd, err := s.Reviews.GetRound(ctx, in.RoundID)
	if err != nil {
		return review.Snapshot{}, err
	}
	if rd.IsClosed() {
		return review.Snapshot{}, review.ErrRoundClosed
	}
	list, _, err := s.Annotations.List(ctx, application.AnnotationFilter{VersionID: in.VersionID, PageSize: 1000})
	if err != nil {
		return review.Snapshot{}, err
	}
	counts := map[string]int{
		"open": 0, "replied": 0, "review": 0, "resolved": 0, "closed": 0,
	}
	for _, a := range list {
		counts[string(a.Status)]++
	}
	snap := review.Snapshot{
		ID:        s.IDs.New(),
		RoundID:   in.RoundID,
		ProjectID: in.ProjectID,
		BoardID:   in.BoardID,
		VersionID: in.VersionID,
		CreatedAt: s.Clock.Now(),
		CreatedBy: in.By,
		Counts:    counts,
	}
	if err := s.Reviews.SaveSnapshot(ctx, snap); err != nil {
		return review.Snapshot{}, err
	}
	if err := rd.AddSnapshot(snap, s.Clock.Now()); err != nil {
		return review.Snapshot{}, err
	}
	if err := s.Reviews.SaveRound(ctx, rd); err != nil {
		return review.Snapshot{}, err
	}
	_ = s.Audit.Log(ctx, "review.snapshot_create", in.By, "snapshot", snap.ID, "", "")
	return snap, nil
}

func (s *ReviewService) SetConclusion(ctx context.Context, in application.SetConclusionInput) (*review.Round, error) {
	rd, err := s.Reviews.GetRound(ctx, in.RoundID)
	if err != nil {
		return nil, err
	}
	expected := rd.Version
	if err := rd.SetConclusion(in.Conclusion, review.Recommendation(in.Recommendation), in.ActorID, s.Clock.Now()); err != nil {
		return nil, err
	}
	if err := s.Reviews.UpdateRound(ctx, rd, expected); err != nil {
		return nil, err
	}
	_ = s.Audit.Log(ctx, "review.conclusion", in.ActorID, "review_round", rd.ID, "", in.Conclusion)
	return rd, nil
}

func (s *ReviewService) CloseRound(ctx context.Context, in application.CloseRoundInput) (*review.Round, error) {
	rd, err := s.Reviews.GetRound(ctx, in.RoundID)
	if err != nil {
		return nil, err
	}
	expected := rd.Version
	if err := rd.Close(s.Clock.Now()); err != nil {
		return nil, err
	}
	if err := s.Reviews.UpdateRound(ctx, rd, expected); err != nil {
		return nil, err
	}
	_ = s.Audit.Log(ctx, "review.round_close", in.ActorID, "review_round", rd.ID, "open", "closed")
	return rd, nil
}

// AuditService reads audit entries.
type AuditService struct {
	Logger audit.Logger
}

func (s *AuditService) Query(ctx context.Context, filter application.AuditFilter) ([]audit.Entry, error) {
	return s.Logger.Query(ctx, audit.Filter{
		ActorID:    filter.ActorID,
		EntityType: filter.EntityType,
		EntityID:   filter.EntityID,
		From:       filter.From,
		To:         filter.To,
	})
}

// NotifierService proxies the offline notifier adapter.
type NotifierService struct {
	N *notify.Notifier
}

func (s *NotifierService) List(userID string, unreadOnly bool) []notify.Message {
	return s.N.List(userID, unreadOnly)
}

func (s *NotifierService) MarkRead(userID, msgID string) bool {
	return s.N.MarkRead(userID, msgID)
}

func (s *NotifierService) UnreadCount(userID string) int {
	return s.N.UnreadCount(userID)
}

// ExportService provides a local-only export of annotation lists and review minutes.
type ExportService struct {
	Annotations application.AnnotationRepository
	Reviews     application.ReviewRepository
	Projects    application.ProjectRepository
}

func (s *ExportService) ExportAnnotationsCSV(ctx context.Context, filter application.AnnotationFilter) (string, error) {
	list, total, err := s.Annotations.List(ctx, filter)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("id,title,status,priority,assignee,reporter,version,created_at,due_at,replies,attachments\n")
	for _, a := range list {
		due := ""
		if a.DueAt != nil {
			due = a.DueAt.Format(time.RFC3339)
		}
		b.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s,%s,%d,%d\n",
			a.ID, csvSafe(a.Title), a.Status, a.Priority, a.AssigneeID, a.ReporterID, a.VersionID,
			a.CreatedAt.Format(time.RFC3339), due, len(a.Replies), len(a.Attachments)))
	}
	_ = total
	return b.String(), nil
}

func (s *ExportService) ExportReviewMinutesCSV(ctx context.Context, roundID string) (string, error) {
	rd, err := s.Reviews.GetRound(ctx, roundID)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("round_id,title,status,conclusion,recommendation,decided_by,decided_at,snapshots\n")
	decidedAt := ""
	if rd.DecidedAt != nil {
		decidedAt = rd.DecidedAt.Format(time.RFC3339)
	}
	b.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%d\n",
		rd.ID, csvSafe(rd.Title), rd.Status, csvSafe(rd.Conclusion), rd.Recommendation,
		rd.DecidedBy, decidedAt, len(rd.Snapshots)))
	return b.String(), nil
}

func csvSafe(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}
