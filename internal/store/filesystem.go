package store

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

type FileBlobStore struct {
	root    string
	counter atomic.Uint64
}

func NewFileBlobStore(root string) (*FileBlobStore, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(absolute, "staging"), 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(absolute, "objects"), 0o755); err != nil {
		return nil, err
	}
	return &FileBlobStore{root: absolute}, nil
}

func (store *FileBlobStore) Stage(ctx context.Context, name string) (StagedBlob, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	sequence := store.counter.Add(1)
	key := fmt.Sprintf("%08d-%s", sequence, filepath.Base(name))
	temporary := filepath.Join(store.root, "staging", key+".part")
	final := filepath.Join(store.root, "objects", key)
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	return &fileStagedBlob{file: file, key: key, temporary: temporary, final: final}, nil
}

func (store *FileBlobStore) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if filepath.Base(key) != key {
		return nil, domainInvalidPath()
	}
	return os.Open(filepath.Join(store.root, "objects", key))
}

func domainInvalidPath() error { return fmt.Errorf("invalid storage key") }

type fileStagedBlob struct {
	mu        sync.Mutex
	file      *os.File
	key       string
	temporary string
	final     string
	closed    bool
	committed bool
}

func (blob *fileStagedBlob) Key() string { return blob.key }

func (blob *fileStagedBlob) Write(data []byte) (int, error) {
	blob.mu.Lock()
	defer blob.mu.Unlock()
	if blob.closed {
		return 0, os.ErrClosed
	}
	return blob.file.Write(data)
}

func (blob *fileStagedBlob) Close() error {
	blob.mu.Lock()
	defer blob.mu.Unlock()
	if blob.closed {
		return nil
	}
	blob.closed = true
	return blob.file.Close()
}

func (blob *fileStagedBlob) Commit(ctx context.Context) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	blob.mu.Lock()
	defer blob.mu.Unlock()
	if blob.committed {
		return nil
	}
	if !blob.closed {
		if err := blob.file.Sync(); err != nil {
			return err
		}
		if err := blob.file.Close(); err != nil {
			return err
		}
		blob.closed = true
	}
	if err := os.Rename(blob.temporary, blob.final); err != nil {
		return err
	}
	blob.committed = true
	return nil
}

func (blob *fileStagedBlob) Abort(context.Context) error {
	blob.mu.Lock()
	defer blob.mu.Unlock()
	if blob.committed {
		return nil
	}
	if !blob.closed {
		_ = blob.file.Close()
		blob.closed = true
	}
	err := os.Remove(blob.temporary)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
