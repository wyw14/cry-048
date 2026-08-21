package store

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

type MemoryExportStore struct {
	mu       sync.RWMutex
	sequence atomic.Uint64
	objects  map[string][]byte
}

func NewMemoryExportStore() *MemoryExportStore {
	return &MemoryExportStore{objects: make(map[string][]byte)}
}

func (store *MemoryExportStore) Begin(ctx context.Context, prefix string) (StagedExport, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	key := fmt.Sprintf("%s-%08d.csv", prefix, store.sequence.Add(1))
	return &memoryStagedExport{store: store, key: key}, nil
}

func (store *MemoryExportStore) Read(key string) ([]byte, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	value, ok := store.objects[key]
	return append([]byte(nil), value...), ok
}

type memoryStagedExport struct {
	mu        sync.Mutex
	store     *MemoryExportStore
	key       string
	buffer    bytes.Buffer
	committed bool
	aborted   bool
}

func (export *memoryStagedExport) Write(data []byte) (int, error) {
	export.mu.Lock()
	defer export.mu.Unlock()
	if export.committed || export.aborted {
		return 0, fmt.Errorf("export closed")
	}
	return export.buffer.Write(data)
}

func (export *memoryStagedExport) Commit(ctx context.Context) (string, error) {
	if err := checkContext(ctx); err != nil {
		return "", err
	}
	export.mu.Lock()
	defer export.mu.Unlock()
	if export.aborted {
		return "", fmt.Errorf("export aborted")
	}
	if export.committed {
		return export.key, nil
	}
	export.store.mu.Lock()
	export.store.objects[export.key] = append([]byte(nil), export.buffer.Bytes()...)
	export.store.mu.Unlock()
	export.committed = true
	return export.key, nil
}

func (export *memoryStagedExport) Abort(context.Context) error {
	export.mu.Lock()
	defer export.mu.Unlock()
	if export.committed {
		return nil
	}
	export.aborted = true
	export.buffer.Reset()
	return nil
}
