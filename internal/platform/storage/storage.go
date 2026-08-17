// Package storage provides a local filesystem-backed attachment store with validation.
package storage

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrEmptyFilename      = errors.New("filename is empty")
	ErrPathTraversal      = errors.New("filename contains path separators")
	ErrForbiddenType      = errors.New("media type is not allowed")
	ErrSizeTooLarge       = errors.New("size exceeds limit")
	ErrSizeMismatch       = errors.New("size does not match content length")
	ErrAttachmentNotFound = errors.New("attachment not found on disk")
)

// Limits for upload validation.
type Limits struct {
	MaxBytes     int64
	AllowedTypes map[string]bool
	BaseDir      string
}

// DefaultLimits returns the production-style limits used by the platform.
func DefaultLimits(baseDir string) Limits {
	return Limits{
		MaxBytes: 20 * 1024 * 1024,
		AllowedTypes: map[string]bool{
			"image/png":       true,
			"image/jpeg":      true,
			"image/webp":      true,
			"image/gif":       true,
			"application/pdf": true,
			"text/plain":      true,
		},
		BaseDir: baseDir,
	}
}

// Store is a local filesystem storage adapter.
type Store struct {
	limits Limits
}

// New returns a Store using the given limits.
func New(limits Limits) (*Store, error) {
	if limits.BaseDir == "" {
		return nil, errors.New("storage base dir required")
	}
	if err := os.MkdirAll(limits.BaseDir, 0o755); err != nil {
		return nil, err
	}
	return &Store{limits: limits}, nil
}

// ValidateUpload checks the media type and size against the limits.
func (s *Store) ValidateUpload(mediaType string, size int64) error {
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	if !s.limits.AllowedTypes[mediaType] {
		return ErrForbiddenType
	}
	if size <= 0 || size > s.limits.MaxBytes {
		return ErrSizeTooLarge
	}
	return nil
}

// Save stores r under a generated key and returns that key.
func (s *Store) Save(ctx context.Context, filename, mediaType string, size int64, r io.Reader) (string, error) {
	if err := contextErr(ctx); err != nil {
		return "", err
	}
	if err := s.ValidateUpload(mediaType, size); err != nil {
		return "", err
	}
	if filename == "" {
		return "", ErrEmptyFilename
	}
	if !safeFilename(filename) {
		return "", ErrPathTraversal
	}
	id := uuid.NewString()
	ext := filepath.Ext(filename)
	key := id + ext
	full := filepath.Join(s.limits.BaseDir, key)
	tmp, err := os.CreateTemp(s.limits.BaseDir, ".upload-*")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	defer func() { _ = tmp.Close(); _ = os.Remove(tmpName) }()
	written, err := copyWithContext(ctx, tmp, r, size)
	if err != nil {
		return "", err
	}
	if written != size {
		return "", ErrSizeMismatch
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := contextErr(ctx); err != nil {
		return "", err
	}
	if err := os.Rename(tmpName, full); err != nil {
		return "", err
	}
	return key, nil
}

func contextErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func safeFilename(filename string) bool {
	if filename == "" || strings.ContainsAny(filename, `/\\`) {
		return false
	}
	base := filepath.Base(filename)
	return base == filename && base != "." && base != ".."
}

func copyWithContext(ctx context.Context, dst io.Writer, src io.Reader, expected int64) (int64, error) {
	if expected <= 0 {
		return 0, ErrSizeTooLarge
	}
	buf := make([]byte, 32*1024)
	var written int64
	for written < expected {
		if err := contextErr(ctx); err != nil {
			return written, err
		}
		remaining := expected - written
		chunk := int64(len(buf))
		if remaining < chunk {
			chunk = remaining
		}
		n, readErr := src.Read(buf[:chunk])
		if n > 0 {
			wn, writeErr := dst.Write(buf[:n])
			written += int64(wn)
			if writeErr != nil {
				return written, writeErr
			}
			if wn != n {
				return written, io.ErrShortWrite
			}
		}
		if readErr != nil {
			if readErr == io.EOF && written == expected {
				break
			}
			return written, readErr
		}
		if n == 0 {
			return written, io.ErrNoProgress
		}
	}
	return written, nil
}

// Open returns the path on disk for a given key for reading.
func (s *Store) Open(ctx context.Context, key string) (string, error) {
	if strings.ContainsAny(key, `/\`) || key == "" {
		return "", ErrPathTraversal
	}
	full := filepath.Join(s.limits.BaseDir, key)
	if _, err := os.Stat(full); err != nil {
		if os.IsNotExist(err) {
			return "", ErrAttachmentNotFound
		}
		return "", err
	}
	return full, nil
}

// Delete removes a stored object.
func (s *Store) Delete(ctx context.Context, key string) error {
	if strings.ContainsAny(key, `/\`) || key == "" {
		return ErrPathTraversal
	}
	full := filepath.Join(s.limits.BaseDir, key)
	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
