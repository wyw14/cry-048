// Package events provides an in-process event bus for offline callbacks.
package events

import (
	"context"
	"sync"
)

// Event is a marker interface for events.
type Event interface {
	EventName() string
}

// Handler handles a single event.
type Handler func(ctx context.Context, e Event) error

// Bus is an in-memory pub/sub adapter.
type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

func New() *Bus { return &Bus{handlers: make(map[string][]Handler)} }

// Subscribe registers a handler for a given event name.
func (b *Bus) Subscribe(name string, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[name] = append(b.handlers[name], h)
}

// Publish dispatches the event to all subscribers in registration order.
// Returns the first error encountered.
func (b *Bus) Publish(ctx context.Context, e Event) error {
	name := e.EventName()
	b.mu.RLock()
	hs := b.handlers[name]
	b.mu.RUnlock()
	for _, h := range hs {
		if err := h(ctx, e); err != nil {
			return err
		}
	}
	return nil
}
