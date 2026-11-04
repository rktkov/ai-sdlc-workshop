package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/rktkov/ytfeed/migrations"
)

func TestOpenHappy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ytfeed.db")
	db, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	version, err := SchemaVersion(context.Background(), db)
	if err != nil {
		t.Fatalf("SchemaVersion: %v", err)
	}
	if version != 3 {
		t.Fatalf("SchemaVersion = %d, want 3", version)
	}

	var journal string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journal); err != nil {
		t.Fatalf("journal_mode: %v", err)
	}
	if journal != "wal" {
		t.Fatalf("journal_mode = %q, want wal", journal)
	}

	var foreignKeys int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatalf("foreign_keys: %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, want 1", foreignKeys)
	}

	var synchronous int
	if err := db.QueryRow("PRAGMA synchronous").Scan(&synchronous); err != nil {
		t.Fatalf("synchronous: %v", err)
	}
	if synchronous != 2 {
		t.Fatalf("synchronous = %d, want 2 (FULL)", synchronous)
	}

	var count int
	if err := db.QueryRow("SELECT count(*) FROM users").Scan(&count); err != nil {
		t.Fatalf("users table: %v", err)
	}
}

func TestOpenUnhappy(t *testing.T) {
	t.Run("directory path", func(t *testing.T) {
		if _, err := Open(context.Background(), t.TempDir()); err == nil {
			t.Fatal("Open(directory) = nil, want an error")
		}
	})

	t.Run("newer schema", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "ytfeed.db")
		db, err := Open(context.Background(), path)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		if _, err := db.Exec("UPDATE schema_migrations SET version = 999 WHERE version = 3"); err != nil {
			t.Fatalf("set newer schema: %v", err)
		}
		_ = db.Close()

		if _, err := Open(context.Background(), path); !errors.Is(err, ErrSchemaNewer) {
			t.Fatalf("Open(newer) = %v, want ErrSchemaNewer", err)
		}
	})
}

func TestOpenCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := Open(ctx, filepath.Join(t.TempDir(), "ytfeed.db")); err == nil {
		t.Fatal("Open(canceled) = nil, want an error")
	}
}

func TestSchemaVersionClosed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ytfeed.db")
	db, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	_ = db.Close()

	if _, err := SchemaVersion(context.Background(), db); err == nil {
		t.Fatal("SchemaVersion(closed) = nil, want an error")
	}
}

func TestBackupDirectory(t *testing.T) {
	if _, err := Backup(t.TempDir()); err == nil {
		t.Fatal("Backup(directory) = nil, want an error")
	}
}

func TestOpenMemory(t *testing.T) {
	db, err := Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:): %v", err)
	}
	defer func() { _ = db.Close() }()

	version, err := SchemaVersion(context.Background(), db)
	if err != nil {
		t.Fatalf("SchemaVersion: %v", err)
	}
	if version != 3 {
		t.Fatalf("SchemaVersion = %d, want 3", version)
	}
}

func TestOpenTwice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ytfeed.db")
	first, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	if _, err := first.Exec("CREATE TABLE marker (id INTEGER)"); err != nil {
		t.Fatalf("write marker: %v", err)
	}
	_ = first.Close()

	second, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer func() { _ = second.Close() }()

	var count int
	if err := second.QueryRow("SELECT count(*) FROM marker").Scan(&count); err != nil {
		t.Fatalf("marker table: %v", err)
	}
}

func TestOpenCreatesBackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ytfeed.db")
	db, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	matches, err := filepath.Glob(path + ".bak-*")
	if err != nil {
		t.Fatalf("glob backups: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("Open did not create a backup")
	}
}

func TestMigrationFiles(t *testing.T) {
	files, latest, err := migrationFiles(migrations.FS)
	if err != nil {
		t.Fatalf("migrationFiles: %v", err)
	}
	if len(files) != 3 || latest != 3 {
		t.Fatalf("files = %v, latest = %d", files, latest)
	}
}

func TestReadUp(t *testing.T) {
	up, err := readUp(migrations.FS, "0001_init.sql")
	if err != nil {
		t.Fatalf("readUp: %v", err)
	}
	if !strings.Contains(up, "CREATE TABLE users") {
		t.Fatalf("up = %q", up)
	}

	if _, err := readUp(migrations.FS, "missing.sql"); err == nil {
		t.Fatal("readUp(missing) = nil, want an error")
	}
}

func TestPruneBackups(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ytfeed.db")
	for _, name := range []string{".bak-1", ".bak-2", ".bak-3"} {
		if err := os.WriteFile(path+name, []byte("x"), 0o600); err != nil {
			t.Fatalf("write backup: %v", err)
		}
	}

	if err := pruneBackups(path, 2); err != nil {
		t.Fatalf("pruneBackups: %v", err)
	}
	matches, err := filepath.Glob(path + ".bak-*")
	if err != nil {
		t.Fatalf("glob backups: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("backups = %v, want 2", matches)
	}
}

func TestPruneBackupsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ytfeed.db")
	backupDir := path + ".bak-1"
	if err := os.Mkdir(backupDir, 0o700); err != nil {
		t.Fatalf("mkdir backup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backupDir, "keep"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write inside backup: %v", err)
	}

	if err := pruneBackups(path, 0); err == nil {
		t.Fatal("pruneBackups(non-empty dir) = nil, want an error")
	}
}

func TestOpenMkdirError(t *testing.T) {
	file := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if _, err := Open(context.Background(), filepath.Join(file, "ytfeed.db")); err == nil {
		t.Fatal("Open(path under file) = nil, want an error")
	}
}

func TestBackupHappy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ytfeed.db")
	if err := os.WriteFile(path, []byte("database-bytes"), 0o600); err != nil {
		t.Fatalf("write db: %v", err)
	}

	copyPath, err := Backup(path)
	if err != nil {
		t.Fatalf("Backup: %v", err)
	}
	raw, err := os.ReadFile(copyPath)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(raw) != "database-bytes" {
		t.Fatalf("backup content = %q", raw)
	}
}

func TestBackupUnhappy(t *testing.T) {
	if _, err := Backup(filepath.Join(t.TempDir(), "missing.db")); err == nil {
		t.Fatal("Backup(missing) = nil, want an error")
	}
}

func TestMigrationFilesInvalid(t *testing.T) {
	fsys := fstest.MapFS{"bad.sql": &fstest.MapFile{Data: []byte("-- no up")}}
	if _, _, err := migrationFiles(fsys); err == nil {
		t.Fatal("migrationFiles(invalid) = nil, want an error")
	}
}

func TestReadUpMissingSection(t *testing.T) {
	fsys := fstest.MapFS{"0001_bad.sql": &fstest.MapFile{Data: []byte("SELECT 1")}}
	if _, err := readUp(fsys, "0001_bad.sql"); err == nil {
		t.Fatal("readUp(no Up) = nil, want an error")
	}
}
