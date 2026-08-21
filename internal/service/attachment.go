package service

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"

	"designreview/internal/domain"
	"designreview/internal/store"
)

type AttachmentService struct {
	repository store.Repository
	blobs      store.BlobStore
	policy     domain.AttachmentPolicy
	clock      domain.Clock
	sequence   atomic.Uint64
}

func NewAttachmentService(repository store.Repository, blobs store.BlobStore, policy domain.AttachmentPolicy, clock domain.Clock) *AttachmentService {
	return &AttachmentService{repository: repository, blobs: blobs, policy: policy, clock: clock}
}

type UploadAttachmentCommand struct {
	ProjectID domain.ID
	Name      string
	MIME      string
	Size      int64
	Body      io.Reader
	Actor     domain.Actor
}

func (service *AttachmentService) Upload(ctx context.Context, command UploadAttachmentCommand) (attachment domain.Attachment, err error) {
	if command.Body == nil {
		return domain.Attachment{}, fmt.Errorf("%w: upload body required", domain.ErrInvalidArgument)
	}
	if err := service.policy.Validate(command.Name, command.MIME, command.Size); err != nil {
		return domain.Attachment{}, err
	}
	if _, err := NewAuthorization(service.repository).RequireEdit(ctx, command.ProjectID, command.Actor); err != nil {
		return domain.Attachment{}, err
	}
	staged, err := service.blobs.Stage(ctx, command.Name)
	if err != nil {
		return domain.Attachment{}, domain.Wrap("stage", "attachment", command.Name, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = staged.Abort(context.WithoutCancel(ctx))
		}
	}()
	written, err := copyWithContext(ctx, staged, io.LimitReader(command.Body, command.Size+1))
	if err != nil {
		return domain.Attachment{}, domain.Wrap("write", "attachment", command.Name, err)
	}
	if written != command.Size {
		return domain.Attachment{}, domain.NewValidationError("invalid attachment", domain.FieldError{Field: "size", Message: "declared size mismatch"})
	}
	if err := staged.Close(); err != nil {
		return domain.Attachment{}, domain.Wrap("close staged", "attachment", command.Name, err)
	}
	identifier := domain.ID(fmt.Sprintf("attachment-%08d", service.sequence.Add(1)))
	attachment, err = domain.NewAttachment(identifier, command.ProjectID, command.Name, command.MIME, command.Size, staged.Key(), service.clock.Now())
	if err != nil {
		return domain.Attachment{}, err
	}
	if err := service.repository.PutAttachment(ctx, attachment); err != nil {
		return domain.Attachment{}, domain.Wrap("save staged", "attachment", identifier.String(), err)
	}
	if err := staged.Commit(ctx); err != nil {
		return domain.Attachment{}, domain.Wrap("publish blob", "attachment", identifier.String(), err)
	}
	committed = true
	attachment, err = attachment.Publish(service.clock.Now())
	if err != nil {
		return domain.Attachment{}, err
	}
	if err := service.repository.PutAttachment(ctx, attachment); err != nil {
		return domain.Attachment{}, domain.Wrap("save published", "attachment", identifier.String(), err)
	}
	return attachment, nil
}

func copyWithContext(ctx context.Context, destination io.Writer, source io.Reader) (int64, error) {
	buffer := make([]byte, 32*1024)
	var total int64
	for {
		select {
		case <-ctx.Done():
			return total, fmt.Errorf("%w: %v", domain.ErrCanceled, ctx.Err())
		default:
		}
		count, readErr := source.Read(buffer)
		if count > 0 {
			written, writeErr := destination.Write(buffer[:count])
			total += int64(written)
			if writeErr != nil {
				return total, writeErr
			}
			if written != count {
				return total, io.ErrShortWrite
			}
		}
		if readErr == io.EOF {
			return total, nil
		}
		if readErr != nil {
			return total, readErr
		}
	}
}
