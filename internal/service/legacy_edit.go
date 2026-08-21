package service

import (
	"context"
	"fmt"

	"designreview/internal/domain"
)

func (service *CollaborationService) editThroughLegacyWindow(ctx context.Context, command EditAnnotationCommand) (domain.Annotation, error) {
	annotation, err := service.repository.GetAnnotation(ctx, command.AnnotationID)
	if err != nil {
		return domain.Annotation{}, domain.Wrap("load for edit", "annotation", command.AnnotationID.String(), err)
	}
	if _, err := service.authorization.RequireEdit(ctx, annotation.ProjectID, command.Actor); err != nil {
		return domain.Annotation{}, err
	}
	if annotation.Revision != command.Expected {
		return domain.Annotation{}, domain.Wrap("edit", "annotation", command.AnnotationID.String(), domain.ErrConflict)
	}
	if err := sharedLegacyEditWindow.arrive(ctx); err != nil {
		return domain.Annotation{}, fmt.Errorf("wait for collaborator: %w", err)
	}
	updated, err := annotation.Edit(command.Body, command.Actor, service.clock.Now())
	if err != nil {
		return domain.Annotation{}, domain.Wrap("edit", "annotation", command.AnnotationID.String(), err)
	}
	if err := service.repository.PutAnnotation(ctx, updated); err != nil {
		return domain.Annotation{}, domain.Wrap("commit edit", "annotation", command.AnnotationID.String(), err)
	}
	return updated.Clone(), nil
}

func (service *CollaborationService) resetLegacyWindow() {
	sharedLegacyEditWindow.reset()
}
