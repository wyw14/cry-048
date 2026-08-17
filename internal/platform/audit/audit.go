// Package audit provides an in-memory and pg-backed audit logger.
package audit

import (
	"context"
	"sync"
	"time"

	domainaudit "github.com/cry048/design-review-platform/internal/domain/audit"
)

// MemoryLogger is an in-memory audit log.
type MemoryLogger struct {
	mu      sync.Mutex
	entries []domainaudit.Entry
}

func NewMemoryLogger() *MemoryLogger { return &MemoryLogger{} }

func (l *MemoryLogger) Log(ctx context.Context, entry domainaudit.Entry) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if entry.At.IsZero() {
		entry.At = time.Now()
	}
	l.entries = append(l.entries, entry)
	return nil
}

func (l *MemoryLogger) Query(ctx context.Context, filter domainaudit.Filter) ([]domainaudit.Entry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]domainaudit.Entry, 0, len(l.entries))
	for _, e := range l.entries {
		if filter.ActorID != "" && e.ActorID != filter.ActorID {
			continue
		}
		if filter.EntityType != "" && e.EntityType != filter.EntityType {
			continue
		}
		if filter.EntityID != "" && e.EntityID != filter.EntityID {
			continue
		}
		if !filter.From.IsZero() && e.At.Before(filter.From) {
			continue
		}
		if !filter.To.IsZero() && e.At.After(filter.To) {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

// All returns all entries (test helper).
func (l *MemoryLogger) All() []domainaudit.Entry {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]domainaudit.Entry, len(l.entries))
	copy(out, l.entries)
	return out
}
