package domain

import (
	"fmt"
	"strings"
	"time"
)

type AnnotationState string

const (
	AnnotationOpen     AnnotationState = "open"
	AnnotationReview   AnnotationState = "review"
	AnnotationResolved AnnotationState = "resolved"
)

type Anchor struct {
	VersionID ID
	NodeKey   string
	X         float64
	Y         float64
}

func (anchor Anchor) Valid() bool {
	return anchor.VersionID.Valid() && strings.TrimSpace(anchor.NodeKey) != "" && anchor.X >= 0 && anchor.Y >= 0
}

type AnnotationRevision struct {
	Number    Revision
	Body      string
	ActorID   ID
	CreatedAt time.Time
}

type Reply struct {
	ID        ID
	Body      string
	ActorID   ID
	CreatedAt time.Time
}

type Annotation struct {
	ID             ID
	ProjectID      ID
	BoardID        ID
	Anchor         Anchor
	Body           string
	State          AnnotationState
	Revision       Revision
	History        []AnnotationRevision
	Replies        []Reply
	MigrationIssue string
}

func NewAnnotation(id, projectID, boardID ID, anchor Anchor, body string, actor Actor, now time.Time) (Annotation, error) {
	body = strings.TrimSpace(body)
	if !id.Valid() || !projectID.Valid() || !boardID.Valid() || !anchor.Valid() || body == "" || !actor.Valid() {
		return Annotation{}, fmt.Errorf("%w: invalid annotation", ErrInvalidArgument)
	}
	return Annotation{
		ID: id, ProjectID: projectID, BoardID: boardID, Anchor: anchor, Body: body,
		State: AnnotationOpen, Revision: 1,
		History: []AnnotationRevision{{Number: 1, Body: body, ActorID: actor.ID, CreatedAt: now}},
	}, nil
}

func (annotation Annotation) Edit(body string, actor Actor, now time.Time) (Annotation, error) {
	body = strings.TrimSpace(body)
	if body == "" || !actor.Valid() {
		return Annotation{}, fmt.Errorf("%w: body and actor are required", ErrInvalidArgument)
	}
	if annotation.State == AnnotationResolved {
		return Annotation{}, fmt.Errorf("%w: resolved annotation cannot be edited", ErrInvalidState)
	}
	result := annotation.Clone()
	result.Body = body
	result.Revision = annotation.Revision.Next()
	result.History = append(result.History, AnnotationRevision{
		Number: result.Revision, Body: body, ActorID: actor.ID, CreatedAt: now,
	})
	return result, nil
}

func (annotation Annotation) Transition(target AnnotationState, actor Actor, now time.Time) (Annotation, error) {
	allowed := map[AnnotationState]map[AnnotationState]bool{
		AnnotationOpen:     {AnnotationReview: true},
		AnnotationReview:   {AnnotationOpen: true, AnnotationResolved: true},
		AnnotationResolved: {AnnotationOpen: true},
	}
	if !allowed[annotation.State][target] || !actor.Valid() {
		return Annotation{}, fmt.Errorf("%w: %s to %s", ErrInvalidState, annotation.State, target)
	}
	result := annotation.Clone()
	result.State = target
	result.Revision = annotation.Revision.Next()
	result.History = append(result.History, AnnotationRevision{
		Number: result.Revision, Body: result.Body, ActorID: actor.ID, CreatedAt: now,
	})
	return result, nil
}

func (annotation Annotation) AddReply(reply Reply) (Annotation, error) {
	if !reply.ID.Valid() || !reply.ActorID.Valid() || strings.TrimSpace(reply.Body) == "" {
		return Annotation{}, fmt.Errorf("%w: invalid reply", ErrInvalidArgument)
	}
	result := annotation.Clone()
	result.Replies = append(result.Replies, reply)
	result.Revision = annotation.Revision.Next()
	return result, nil
}

func (annotation Annotation) Clone() Annotation {
	clone := annotation
	clone.History = append([]AnnotationRevision(nil), annotation.History...)
	clone.Replies = append([]Reply(nil), annotation.Replies...)
	return clone
}
