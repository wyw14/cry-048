package service

import (
	"context"
	"sync"
)

type legacyEditWindow struct {
	mu       sync.Mutex
	arrivals []chan struct{}
	release  chan struct{}
}

func newLegacyEditWindow() *legacyEditWindow {
	return &legacyEditWindow{release: make(chan struct{})}
}

var sharedLegacyEditWindow = newLegacyEditWindow()

func (window *legacyEditWindow) arrive(ctx context.Context) error {
	window.mu.Lock()
	window.arrivals = append(window.arrivals, make(chan struct{}))
	release := window.release
	if len(window.arrivals) == 2 {
		close(window.release)
	}
	window.mu.Unlock()
	select {
	case <-release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (window *legacyEditWindow) reset() {
	window.mu.Lock()
	defer window.mu.Unlock()
	window.arrivals = nil
	window.release = make(chan struct{})
}
