package domain

import (
	"fmt"
	"mime"
	"path/filepath"
	"strings"
	"time"
)

type AttachmentState string

const (
	AttachmentStaged    AttachmentState = "staged"
	AttachmentPublished AttachmentState = "published"
	AttachmentFailed    AttachmentState = "failed"
)

type Attachment struct {
	ID          ID
	ProjectID   ID
	Name        string
	MIME        string
	Size        int64
	StorageKey  string
	State       AttachmentState
	CreatedAt   time.Time
	PublishedAt *time.Time
}

type AttachmentPolicy struct {
	AllowedMIME map[string]struct{}
	MaxBytes    int64
}

func (policy AttachmentPolicy) Validate(name, contentType string, size int64) error {
	name = strings.TrimSpace(name)
	if name == "" || filepath.IsAbs(name) || strings.Contains(name, "..") || strings.ContainsAny(name, "\\/") {
		return NewValidationError("invalid attachment", FieldError{Field: "name", Message: "unsafe path"})
	}
	cleanMIME, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return NewValidationError("invalid attachment", FieldError{Field: "mime", Message: "invalid media type"})
	}
	if _, allowed := policy.AllowedMIME[cleanMIME]; !allowed {
		return NewValidationError("invalid attachment", FieldError{Field: "mime", Message: "not allowed"})
	}
	if size < 0 || size > policy.MaxBytes {
		return NewValidationError("invalid attachment", FieldError{Field: "size", Message: "limit exceeded"})
	}
	return nil
}

func NewAttachment(id, projectID ID, name, contentType string, size int64, key string, now time.Time) (Attachment, error) {
	if !id.Valid() || !projectID.Valid() || key == "" {
		return Attachment{}, fmt.Errorf("%w: invalid attachment identity", ErrInvalidArgument)
	}
	return Attachment{ID: id, ProjectID: projectID, Name: name, MIME: contentType, Size: size, StorageKey: key, State: AttachmentStaged, CreatedAt: now}, nil
}

func (attachment Attachment) Publish(now time.Time) (Attachment, error) {
	if attachment.State != AttachmentStaged {
		return Attachment{}, fmt.Errorf("%w: attachment is not staged", ErrInvalidState)
	}
	result := attachment
	result.State = AttachmentPublished
	result.PublishedAt = &now
	return result, nil
}
