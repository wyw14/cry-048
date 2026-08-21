package service

import (
	"context"
	"fmt"

	"designreview/internal/domain"
	"designreview/internal/store"
)

type CollaborationService struct {
	repository    store.Repository
	authorization *Authorization
	clock         domain.Clock
}

func NewCollaborationService(repository store.Repository, clock domain.Clock) *CollaborationService {
	return &CollaborationService{repository: repository, authorization: NewAuthorization(repository), clock: clock}
}

type EditAnnotationCommand struct {
	AnnotationID domain.ID
	Expected     domain.Revision
	Body         string
	Actor        domain.Actor
}

func (service *CollaborationService) EditAnnotation(ctx context.Context, command EditAnnotationCommand) (domain.Annotation, error) {
	return service.editThroughLegacyWindow(ctx, command)
}

func (service *CollaborationService) editAnnotationAtomic(ctx context.Context, command EditAnnotationCommand) (domain.Annotation, error) {
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
	updated, err := annotation.Edit(command.Body, command.Actor, service.clock.Now())
	if err != nil {
		return domain.Annotation{}, domain.Wrap("edit", "annotation", command.AnnotationID.String(), err)
	}
	if err := service.repository.CompareAndSwapAnnotation(ctx, updated, command.Expected); err != nil {
		return domain.Annotation{}, domain.Wrap("commit edit", "annotation", command.AnnotationID.String(), err)
	}
	return updated.Clone(), nil
}

type TransitionAnnotationCommand struct {
	AnnotationID domain.ID
	Expected     domain.Revision
	Target       domain.AnnotationState
	Actor        domain.Actor
}

func (service *CollaborationService) TransitionAnnotation(ctx context.Context, command TransitionAnnotationCommand) (domain.Annotation, error) {
	annotation, err := service.repository.GetAnnotation(ctx, command.AnnotationID)
	if err != nil {
		return domain.Annotation{}, domain.Wrap("load for transition", "annotation", command.AnnotationID.String(), err)
	}
	if _, err := service.authorization.RequireReview(ctx, annotation.ProjectID, command.Actor); err != nil {
		return domain.Annotation{}, err
	}
	if annotation.Revision != command.Expected {
		return domain.Annotation{}, domain.ErrConflict
	}
	updated, err := annotation.Transition(command.Target, command.Actor, service.clock.Now())
	if err != nil {
		return domain.Annotation{}, domain.Wrap("transition", "annotation", command.AnnotationID.String(), err)
	}
	if err := service.repository.CompareAndSwapAnnotation(ctx, updated, command.Expected); err != nil {
		return domain.Annotation{}, domain.Wrap("commit transition", "annotation", command.AnnotationID.String(), err)
	}
	return updated.Clone(), nil
}

func (service *CollaborationService) AddReply(ctx context.Context, annotationID domain.ID, expected domain.Revision, reply domain.Reply, actor domain.Actor) (domain.Annotation, error) {
	annotation, err := service.repository.GetAnnotation(ctx, annotationID)
	if err != nil {
		return domain.Annotation{}, domain.Wrap("load for reply", "annotation", annotationID.String(), err)
	}
	if annotation.Revision != expected {
		return domain.Annotation{}, domain.ErrConflict
	}
	if reply.ActorID != actor.ID {
		return domain.Annotation{}, fmt.Errorf("%w: reply actor mismatch", domain.ErrForbidden)
	}
	if _, err := service.authorization.RequireReview(ctx, annotation.ProjectID, actor); err != nil {
		return domain.Annotation{}, err
	}
	updated, err := annotation.AddReply(reply)
	if err != nil {
		return domain.Annotation{}, err
	}
	if err := service.repository.CompareAndSwapAnnotation(ctx, updated, expected); err != nil {
		return domain.Annotation{}, domain.Wrap("commit reply", "annotation", annotationID.String(), err)
	}
	return updated.Clone(), nil
}
