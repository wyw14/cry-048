// Package review defines review rounds, snapshots, summaries, and recommendations.
package review

import (
	"errors"
	"time"
)

var (
	ErrRoundNotFound         = errors.New("review round not found")
	ErrRoundClosed           = errors.New("review round is closed")
	ErrRoundAlreadyClosed    = errors.New("review round already closed")
	ErrSnapshotNotFound      = errors.New("snapshot not found")
	ErrSummaryEmpty          = errors.New("summary conclusion cannot be empty")
	ErrInvalidRecommendation = errors.New("invalid recommendation")
)

// Recommendation is the publish decision.
type Recommendation string

const (
	RecApprove               Recommendation = "approve"
	RecApproveWithConditions Recommendation = "approve_with_conditions"
	RecReject                Recommendation = "reject"
	RecDefer                 Recommendation = "defer"
)

func (r Recommendation) Valid() bool {
	switch r {
	case RecApprove, RecApproveWithConditions, RecReject, RecDefer:
		return true
	}
	return false
}

// Snapshot is a frozen point-in-time capture of all open annotations for a board/version.
type Snapshot struct {
	ID        string
	RoundID   string
	ProjectID string
	BoardID   string
	VersionID string
	CreatedAt time.Time
	CreatedBy string
	Counts    map[string]int
}

// Round is one review cycle.
type Round struct {
	ID             string
	ProjectID      string
	BoardID        string
	Title          string
	Status         string // open | closed
	Snapshots      []Snapshot
	Conclusion     string
	Recommendation Recommendation
	DecidedBy      string
	DecidedAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Version        int
	ClosedAt       *time.Time
}

func NewRound(id, projectID, boardID, title string, now time.Time) (*Round, error) {
	if id == "" || projectID == "" || boardID == "" {
		return nil, errors.New("round id, project id, and board id required")
	}
	if title == "" {
		return nil, errors.New("round title required")
	}
	return &Round{
		ID:        id,
		ProjectID: projectID,
		BoardID:   boardID,
		Title:     title,
		Status:    "open",
		CreatedAt: now,
		UpdatedAt: now,
		Version:   1,
	}, nil
}

// AddSnapshot appends a snapshot to the round; only allowed while open.
func (r *Round) AddSnapshot(s Snapshot, now time.Time) error {
	if r.Status == "closed" {
		return ErrRoundClosed
	}
	s.RoundID = r.ID
	r.Snapshots = append(r.Snapshots, s)
	r.UpdatedAt = now
	r.Version++
	return nil
}

// SetConclusion sets the summary conclusion and recommendation; only while open.
func (r *Round) SetConclusion(conclusion string, rec Recommendation, by string, now time.Time) error {
	if r.Status == "closed" {
		return ErrRoundClosed
	}
	if conclusion == "" {
		return ErrSummaryEmpty
	}
	if !rec.Valid() {
		return ErrInvalidRecommendation
	}
	if by == "" {
		return errors.New("decider required")
	}
	r.Conclusion = conclusion
	r.Recommendation = rec
	r.DecidedBy = by
	decidedAt := now
	r.DecidedAt = &decidedAt
	r.UpdatedAt = now
	r.Version++
	return nil
}

// Close finalizes the round.
func (r *Round) Close(now time.Time) error {
	if r.Status == "closed" {
		return ErrRoundAlreadyClosed
	}
	r.Status = "closed"
	closedAt := now
	r.ClosedAt = &closedAt
	r.UpdatedAt = now
	r.Version++
	return nil
}

// IsClosed reports whether the round is finalised.
func (r *Round) IsClosed() bool { return r.Status == "closed" }
