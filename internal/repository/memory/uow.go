package memory

import (
	"context"
	"sync"
)

// MemoryTxRunner emulates transactions by holding a process-wide mutex during fn.
type MemoryTxRunner struct {
	mu sync.Mutex
}

func NewTxRunner() *MemoryTxRunner { return &MemoryTxRunner{} }

func (r *MemoryTxRunner) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return fn(ctx)
}
