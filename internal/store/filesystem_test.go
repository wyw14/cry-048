package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileBlobAbortRemovesTemporaryResource(t *testing.T) {
	root := t.TempDir()
	blobs, err := NewFileBlobStore(root)
	if err != nil {
		t.Fatal(err)
	}
	staged, err := blobs.Stage(context.Background(), "image.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := staged.Write([]byte("payload")); err != nil {
		t.Fatal(err)
	}
	if err := staged.Abort(context.Background()); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(root, "staging"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("staging files remain: %d", len(entries))
	}
}
