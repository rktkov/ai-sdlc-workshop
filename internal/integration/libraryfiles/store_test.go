package libraryfiles

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rktkov/ytfeed/internal/domain/library"
)

func TestPathAndDir(t *testing.T) {
	store := New("/data")
	if store.Dir() != "/data/library" {
		t.Fatalf("Dir = %q", store.Dir())
	}
	if store.Path("v1", "mp4") != "/data/library/v1.mp4" {
		t.Fatalf("Path = %q", store.Path("v1", "mp4"))
	}
}

func TestStatDelete(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)
	if err := store.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}
	path := store.Path("dQw4w9WgXcQ", "mp4")
	if err := os.WriteFile(path, []byte("media"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	bytes, err := store.Stat("dQw4w9WgXcQ", "mp4")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if bytes != 5 {
		t.Fatalf("bytes = %d", bytes)
	}
	if err := store.Delete("dQw4w9WgXcQ", "mp4"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Stat("dQw4w9WgXcQ", "mp4"); !errors.Is(err, library.ErrNotFound) {
		t.Fatalf("Stat = %v, want ErrNotFound", err)
	}
	if err := store.Delete("dQw4w9WgXcQ", "mp4"); err != nil {
		t.Fatalf("Delete missing: %v", err)
	}
}

func TestInvalidNames(t *testing.T) {
	store := New(t.TempDir())
	if _, err := store.Stat("../x", "mp4"); err == nil {
		t.Fatal("Stat(path traversal) = nil, want an error")
	}
	if _, err := store.Stat("v1", "../mp4"); err == nil {
		t.Fatal("Stat(bad extension) = nil, want an error")
	}
	if err := store.Delete("../x", "mp4"); err == nil {
		t.Fatal("Delete(path traversal) = nil, want an error")
	}
}

func TestEnsureDirPermissions(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)
	if err := store.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, "library"))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
}
