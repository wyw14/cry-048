package storage

import (
	"bytes"
	"context"
	"testing"
)

func TestSaveRejectsAlreadyCancelledContext(t *testing.T) {
	dir := t.TempDir()
	store, err := New(Limits{BaseDir: dir, MaxBytes: 1024, AllowedTypes: map[string]bool{"text/plain": true}})
	if err != nil { t.Fatal(err) }
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store.Save(ctx, "note.txt", "text/plain", 4, bytes.NewReader([]byte("data"))); err != context.Canceled {
		t.Fatalf("expected cancellation before creating file, got %v", err)
	}
}
