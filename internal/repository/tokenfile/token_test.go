package tokenfile

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/identity"
)

func TestWriteRead(t *testing.T) {
	store := New(t.TempDir())
	want := identity.Token{
		AccessToken:  "access-1",
		RefreshToken: "refresh-1",
		TokenType:    "Bearer",
		Scope:        identity.ScopeYouTubeReadOnly,
		ExpiresAt:    time.Date(2026, 11, 4, 13, 0, 0, 0, time.UTC),
	}
	if err := store.Write(context.Background(), want); err != nil {
		t.Fatalf("Write: %v", err)
	}

	info, err := os.Stat(store.Path())
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}

	got, err := store.Read(context.Background())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.AccessToken != want.AccessToken || got.RefreshToken != want.RefreshToken {
		t.Fatalf("token = %+v", got)
	}
	if !got.ExpiresAt.Equal(want.ExpiresAt) {
		t.Fatalf("expiry = %v", got.ExpiresAt)
	}
}

func TestReadMissing(t *testing.T) {
	store := New(t.TempDir())
	if _, err := store.Read(context.Background()); !errors.Is(err, identity.ErrTokenNotFound) {
		t.Fatalf("Read = %v, want ErrTokenNotFound", err)
	}
}

func TestCheckPermissions(t *testing.T) {
	store := New(t.TempDir())
	if err := store.Write(context.Background(), identity.Token{AccessToken: "access"}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := store.CheckPermissions(); err != nil {
		t.Fatalf("CheckPermissions: %v", err)
	}
	if err := os.Chmod(store.Path(), 0o644); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	if err := store.CheckPermissions(); !errors.Is(err, identity.ErrInsecurePermissions) {
		t.Fatalf("CheckPermissions = %v, want ErrInsecurePermissions", err)
	}
}

func TestDelete(t *testing.T) {
	store := New(t.TempDir())
	if err := store.Write(context.Background(), identity.Token{AccessToken: "access"}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := store.Delete(context.Background()); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Read(context.Background()); !errors.Is(err, identity.ErrTokenNotFound) {
		t.Fatalf("Read after delete = %v", err)
	}
}

func TestWriteRejectsEmptyToken(t *testing.T) {
	store := New(t.TempDir())
	if err := store.Write(context.Background(), identity.Token{}); err == nil {
		t.Fatal("Write(empty) = nil, want an error")
	}
}

func TestWriteReplacesExisting(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)
	if err := store.Write(context.Background(), identity.Token{AccessToken: "old"}); err != nil {
		t.Fatalf("Write old: %v", err)
	}
	if err := store.Write(context.Background(), identity.Token{AccessToken: "new"}); err != nil {
		t.Fatalf("Write new: %v", err)
	}
	got, err := store.Read(context.Background())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.AccessToken != "new" {
		t.Fatalf("access token = %q", got.AccessToken)
	}
	entries, err := os.ReadDir(filepath.Dir(store.Path()))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("directory has %d entries, want 1", len(entries))
	}
}

func TestReadCorrupt(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "token"), []byte(`{`), 0o600); err != nil {
		t.Fatalf("write corrupt token: %v", err)
	}
	if _, err := New(dir).Read(context.Background()); err == nil {
		t.Fatal("Read(corrupt) = nil, want an error")
	}
}

func TestReadEmptyAccessToken(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "token"), []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write empty token: %v", err)
	}
	if _, err := New(dir).Read(context.Background()); err == nil {
		t.Fatal("Read(empty) = nil, want an error")
	}
}

func TestDeleteMissing(t *testing.T) {
	if err := New(t.TempDir()).Delete(context.Background()); err != nil {
		t.Fatalf("Delete(missing) = %v", err)
	}
}

func TestCheckPermissionsMissing(t *testing.T) {
	if err := New(t.TempDir()).CheckPermissions(); !errors.Is(err, identity.ErrTokenNotFound) {
		t.Fatalf("CheckPermissions = %v, want ErrTokenNotFound", err)
	}
}

func TestWriteDirectoryError(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := New(filepath.Join(file, "data")).Write(context.Background(), identity.Token{AccessToken: "access"}); err == nil {
		t.Fatal("Write = nil, want an error")
	}
}

func TestPath(t *testing.T) {
	store := New("/tmp/ytfeed-test")
	if store.Path() != "/tmp/ytfeed-test/token" {
		t.Fatalf("Path = %q", store.Path())
	}
}

func TestReadIsDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "token"), 0o700); err != nil {
		t.Fatalf("mkdir token: %v", err)
	}
	if _, err := New(dir).Read(context.Background()); err == nil {
		t.Fatal("Read(directory) = nil, want an error")
	}
}

func TestWriteRenameError(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "token"), 0o700); err != nil {
		t.Fatalf("mkdir token: %v", err)
	}
	if err := New(dir).Write(context.Background(), identity.Token{AccessToken: "access"}); err == nil {
		t.Fatal("Write over a directory = nil, want an error")
	}
}
