package transcripts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/analysis"
)

const videoID = "dQw4w9WgXcQ"

func TestPutGet(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)
	text := "hello transcript"

	staged, err := store.Put(context.Background(), videoID, text)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	sum := sha256.Sum256([]byte(text))
	wantHash := hex.EncodeToString(sum[:])
	if staged.Hash != wantHash || staged.Bytes != int64(len(text)) {
		t.Fatalf("staged = %+v", staged)
	}

	info, err := os.Stat(filepath.Join(dir, "tmp", "transcripts", videoID+".txt"))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("file mode = %o, want 600", info.Mode().Perm())
	}
	dirInfo, err := os.Stat(filepath.Join(dir, "tmp", "transcripts"))
	if err != nil {
		t.Fatalf("Stat dir: %v", err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("dir mode = %o, want 700", dirInfo.Mode().Perm())
	}

	got, err := store.Get(context.Background(), videoID, wantHash)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Text != text || got.Hash != wantHash {
		t.Fatalf("got = %+v", got)
	}
}

func TestPutInvalidVideoID(t *testing.T) {
	store := New(t.TempDir())
	for _, id := range []string{"", "short", "../etc/passwd", "dQw4w9WgXc!"} {
		if _, err := store.Put(context.Background(), id, "text"); !errors.Is(err, analysis.ErrInvalidVideoID) {
			t.Fatalf("Put(%q) = %v, want ErrInvalidVideoID", id, err)
		}
	}
}

func TestPutTooLarge(t *testing.T) {
	store := NewWithLimit(t.TempDir(), 4)
	if _, err := store.Put(context.Background(), videoID, "12345"); !errors.Is(err, analysis.ErrTooLarge) {
		t.Fatalf("Put = %v, want ErrTooLarge", err)
	}
}

func TestGetMiss(t *testing.T) {
	store := New(t.TempDir())
	if _, err := store.Get(context.Background(), videoID, ""); !errors.Is(err, analysis.ErrStageMiss) {
		t.Fatalf("Get(missing) = %v, want ErrStageMiss", err)
	}
	if _, err := store.Put(context.Background(), videoID, "text"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if _, err := store.Get(context.Background(), videoID, "wrong"); !errors.Is(err, analysis.ErrStageMiss) {
		t.Fatalf("Get(hash mismatch) = %v, want ErrStageMiss", err)
	}
}

func TestGetSymlink(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)
	if _, err := store.Put(context.Background(), videoID, "text"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	path := filepath.Join(dir, "tmp", "transcripts", videoID+".txt")
	if err := os.Remove(path); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("target"), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatalf("Symlink: %v", err)
	}
	if _, err := store.Get(context.Background(), videoID, ""); !errors.Is(err, analysis.ErrStageMiss) {
		t.Fatalf("Get(symlink) = %v, want ErrStageMiss", err)
	}
}

func TestDelete(t *testing.T) {
	store := New(t.TempDir())
	if err := store.Delete(context.Background(), videoID); err != nil {
		t.Fatalf("Delete(missing) = %v", err)
	}
	if _, err := store.Put(context.Background(), videoID, "text"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := store.Delete(context.Background(), videoID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Get(context.Background(), videoID, ""); !errors.Is(err, analysis.ErrStageMiss) {
		t.Fatalf("Get after delete = %v", err)
	}
}

func TestSweep(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)
	if _, err := store.Put(context.Background(), videoID, "old"); err != nil {
		t.Fatalf("Put old: %v", err)
	}
	oldPath := filepath.Join(dir, "tmp", "transcripts", videoID+".txt")
	oldTime := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(oldPath, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}
	if _, err := store.Put(context.Background(), "abcdefghijk", "new"); err != nil {
		t.Fatalf("Put new: %v", err)
	}
	count, err := store.Sweep(context.Background(), time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	if _, err := os.Stat(oldPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old file still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "tmp", "transcripts", "abcdefghijk.txt")); err != nil {
		t.Fatalf("new file missing: %v", err)
	}
}

func TestSweepMissingDirectory(t *testing.T) {
	store := New(t.TempDir())
	count, err := store.Sweep(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if count != 0 {
		t.Fatalf("count = %d, want 0", count)
	}
}

func TestGetInvalidID(t *testing.T) {
	if _, err := New(t.TempDir()).Get(context.Background(), "short", ""); !errors.Is(err, analysis.ErrInvalidVideoID) {
		t.Fatalf("Get = %v, want ErrInvalidVideoID", err)
	}
}

func TestDeleteInvalidID(t *testing.T) {
	if err := New(t.TempDir()).Delete(context.Background(), "short"); !errors.Is(err, analysis.ErrInvalidVideoID) {
		t.Fatalf("Delete = %v, want ErrInvalidVideoID", err)
	}
}

func TestGetDirectory(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)
	path := filepath.Join(dir, "tmp", "transcripts", videoID+".txt")
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if _, err := store.Get(context.Background(), videoID, ""); !errors.Is(err, analysis.ErrStageMiss) {
		t.Fatalf("Get(directory) = %v, want ErrStageMiss", err)
	}
}

func TestPutReplaces(t *testing.T) {
	store := New(t.TempDir())
	if _, err := store.Put(context.Background(), videoID, "old"); err != nil {
		t.Fatalf("Put old: %v", err)
	}
	if _, err := store.Put(context.Background(), videoID, "new"); err != nil {
		t.Fatalf("Put new: %v", err)
	}
	got, err := store.Get(context.Background(), videoID, "")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Text != "new" {
		t.Fatalf("text = %q", got.Text)
	}
}

func TestPutDirectoryError(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "tmp")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := New(dir).Put(context.Background(), videoID, "text"); err == nil {
		t.Fatal("Put = nil, want an error")
	}
}

func TestSweepSkipsOtherEntries(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)
	if err := os.MkdirAll(store.Dir(), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	old := time.Now().Add(-48 * time.Hour)
	for _, name := range []string{"keep.log", "readme"} {
		path := filepath.Join(store.Dir(), name)
		if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		if err := os.Chtimes(path, old, old); err != nil {
			t.Fatalf("Chtimes: %v", err)
		}
	}
	if err := os.Mkdir(filepath.Join(store.Dir(), "subdir"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	count, err := store.Sweep(context.Background(), time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if count != 0 {
		t.Fatalf("count = %d, want 0", count)
	}
}

func TestSweepPart(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)
	if err := os.MkdirAll(store.Dir(), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	part := filepath.Join(store.Dir(), videoID+".123.part")
	if err := os.WriteFile(part, []byte("x"), 0o600); err != nil {
		t.Fatalf("write part: %v", err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(part, old, old); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}
	count, err := store.Sweep(context.Background(), time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}

func TestGetTooLarge(t *testing.T) {
	dir := t.TempDir()
	store := NewWithLimit(dir, 4)
	if err := os.MkdirAll(store.Dir(), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.Dir(), videoID+".txt"), []byte("12345"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := store.Get(context.Background(), videoID, ""); !errors.Is(err, analysis.ErrStageMiss) {
		t.Fatalf("Get = %v, want ErrStageMiss", err)
	}
}

func TestDeleteDirectory(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)
	path := filepath.Join(store.Dir(), videoID+".txt")
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(path, "keep"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := store.Delete(context.Background(), videoID); err == nil {
		t.Fatal("Delete(non-empty directory) = nil, want an error")
	}
}

func TestPutEmpty(t *testing.T) {
	store := New(t.TempDir())
	staged, err := store.Put(context.Background(), videoID, "")
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if staged.Bytes != 0 {
		t.Fatalf("bytes = %d", staged.Bytes)
	}
}
