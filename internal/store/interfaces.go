package store

import (
	"context"
	"io"
	"time"

	"designreview/internal/domain"
)

type Repository interface {
	GetProject(context.Context, domain.ID) (domain.Project, error)
	PutProject(context.Context, domain.Project) error
	GetBoard(context.Context, domain.ID) (domain.Board, error)
	PutBoard(context.Context, domain.Board) error
	GetVersion(context.Context, domain.ID) (domain.DesignVersion, error)
	ListVersionsByBoard(context.Context, domain.ID) ([]domain.DesignVersion, error)
	PutVersion(context.Context, domain.DesignVersion) error
	GetAnnotation(context.Context, domain.ID) (domain.Annotation, error)
	ListAnnotationsByVersion(context.Context, domain.ID) ([]domain.Annotation, error)
	CompareAndSwapAnnotation(context.Context, domain.Annotation, domain.Revision) error
	PutAnnotation(context.Context, domain.Annotation) error
	GetReviewRound(context.Context, domain.ID) (domain.ReviewRound, error)
	PutReviewRound(context.Context, domain.ReviewRound) error
	GetMember(context.Context, domain.ID, domain.ID) (domain.Member, error)
	FindMemberByEmail(context.Context, domain.ID, string) (domain.Member, error)
	CompareAndSwapMember(context.Context, domain.Member, domain.Revision) error
	PutMember(context.Context, domain.Member) error
	PutAttachment(context.Context, domain.Attachment) error
	AppendAudit(context.Context, domain.AuditEvent) error
	PutNotificationIfAbsent(context.Context, domain.Notification) (bool, error)
	LeaseNotifications(context.Context, string, int, time.Time) ([]domain.Notification, error)
	MarkNotificationSent(context.Context, domain.ID, string, time.Time) error
}

type UnitOfWork interface {
	Within(context.Context, func(context.Context, Repository) error) error
}

type StagedBlob interface {
	io.WriteCloser
	Key() string
	Commit(context.Context) error
	Abort(context.Context) error
}

type BlobStore interface {
	Stage(context.Context, string) (StagedBlob, error)
	Open(context.Context, string) (io.ReadCloser, error)
}

type StagedExport interface {
	io.Writer
	Commit(context.Context) (string, error)
	Abort(context.Context) error
}

type ExportStore interface {
	Begin(context.Context, string) (StagedExport, error)
}
