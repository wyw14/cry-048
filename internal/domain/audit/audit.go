// Package audit defines audit trail entries.
package audit

import (
	"context"
	"time"
)

// Entry is one audit log row.
type Entry struct {
	ID         string
	ActorID    string
	Action     string
	EntityType string
	EntityID   string
	Before     string
	After      string
	At         time.Time
	RequestID  string
}

// Filter for querying audit entries.
type Filter struct {
	ActorID    string
	EntityType string
	EntityID   string
	From       time.Time
	To         time.Time
}

// Logger is the audit writer interface used by application services.
type Logger interface {
	Log(ctx context.Context, entry Entry) error
	Query(ctx context.Context, filter Filter) ([]Entry, error)
}
