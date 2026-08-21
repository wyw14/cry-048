package domain

import (
	"fmt"
	"strings"
	"time"
)

type ReviewStatus string

const (
	ReviewActive ReviewStatus = "active"
	ReviewClosed ReviewStatus = "closed"
)

type ReviewSnapshot struct {
	VersionID       ID
	VersionRevision Revision
	AnnotationIDs   []ID
	CapturedAt      time.Time
}

func (snapshot ReviewSnapshot) Clone() ReviewSnapshot {
	clone := snapshot
	clone.AnnotationIDs = append([]ID(nil), snapshot.AnnotationIDs...)
	return clone
}

type ReviewRound struct {
	ID         ID
	ProjectID  ID
	VersionID  ID
	Status     ReviewStatus
	Conclusion string
	Snapshot   *ReviewSnapshot
	StartedAt  time.Time
	ClosedAt   *time.Time
	ClosedBy   ID
	Revision   Revision
}

func NewReviewRound(id, projectID, versionID ID, now time.Time) (ReviewRound, error) {
	if !id.Valid() || !projectID.Valid() || !versionID.Valid() {
		return ReviewRound{}, fmt.Errorf("%w: invalid review round", ErrInvalidArgument)
	}
	return ReviewRound{ID: id, ProjectID: projectID, VersionID: versionID, Status: ReviewActive, StartedAt: now, Revision: 1}, nil
}

func (round ReviewRound) Close(conclusion string, snapshot ReviewSnapshot, actor Actor, now time.Time) (ReviewRound, error) {
	conclusion = strings.TrimSpace(conclusion)
	if round.Status != ReviewActive {
		return ReviewRound{}, fmt.Errorf("%w: review already closed", ErrInvalidState)
	}
	if conclusion == "" || snapshot.VersionID != round.VersionID || !actor.Valid() {
		return ReviewRound{}, fmt.Errorf("%w: conclusion, snapshot and actor are required", ErrInvalidArgument)
	}
	result := round
	copy := snapshot.Clone()
	result.Status = ReviewClosed
	result.Conclusion = conclusion
	result.Snapshot = &copy
	result.ClosedAt = &now
	result.ClosedBy = actor.ID
	result.Revision = round.Revision.Next()
	return result, nil
}

func (round ReviewRound) Clone() ReviewRound {
	clone := round
	if round.Snapshot != nil {
		copy := round.Snapshot.Clone()
		clone.Snapshot = &copy
	}
	return clone
}
