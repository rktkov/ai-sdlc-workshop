// Package storage opens the SQLite database and applies embedded migrations.
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rktkov/ytfeed/migrations"

	_ "modernc.org/sqlite"
)

// ErrSchemaNewer means the database was written by a newer binary.
var ErrSchemaNewer = errors.New("database schema is newer than this binary")

// Open opens the SQLite database, applies pragmas and runs pending migrations.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	for _, pragma := range []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA synchronous = FULL",
	} {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("%s: %w", pragma, err)
		}
	}

	if err := migrate(ctx, db, path); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// SchemaVersion returns the highest applied migration.
func SchemaVersion(ctx context.Context, db *sql.DB) (int, error) {
	var version int
	err := db.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("read schema version: %w", err)
	}
	return version, nil
}

// Backup copies the database file and returns the backup path.
func Backup(path string) (string, error) {
	source, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open database for backup: %w", err)
	}
	defer func() { _ = source.Close() }()

	copyPath := path + ".bak-" + time.Now().UTC().Format("20060102T150405Z")
	target, err := os.OpenFile(copyPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", fmt.Errorf("create backup: %w", err)
	}
	if _, err := io.Copy(target, source); err != nil {
		_ = target.Close()
		return "", fmt.Errorf("copy backup: %w", err)
	}
	if err := target.Close(); err != nil {
		return "", fmt.Errorf("close backup: %w", err)
	}
	return copyPath, nil
}

type migrationFile struct {
	version int
	name    string
}

func migrate(ctx context.Context, db *sql.DB, path string) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at DATETIME NOT NULL
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	current, err := SchemaVersion(ctx, db)
	if err != nil {
		return err
	}

	files, latest, err := migrationFiles(migrations.FS)
	if err != nil {
		return err
	}
	if current > latest {
		return ErrSchemaNewer
	}

	pending := false
	for _, file := range files {
		if file.version > current {
			pending = true
			break
		}
	}
	if !pending {
		return nil
	}

	if path != ":memory:" {
		if _, err := Backup(path); err != nil {
			return err
		}
		if err := pruneBackups(path, 5); err != nil {
			return err
		}
	}

	for _, file := range files {
		if file.version <= current {
			continue
		}
		up, err := readUp(migrations.FS, file.name)
		if err != nil {
			return err
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", file.version, err)
		}
		if _, err := tx.ExecContext(ctx, up); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %d: %w", file.version, err)
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations(version, applied_at) VALUES (?, ?)", file.version, time.Now().UTC()); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %d: %w", file.version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", file.version, err)
		}
	}
	return nil
}

func migrationFiles(fsys fs.FS) ([]migrationFile, int, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, 0, fmt.Errorf("read migrations: %w", err)
	}

	var files []migrationFile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		prefix := strings.SplitN(entry.Name(), "_", 2)[0]
		version, err := strconv.Atoi(prefix)
		if err != nil {
			return nil, 0, fmt.Errorf("migration %s has no numeric prefix", entry.Name())
		}
		files = append(files, migrationFile{version: version, name: entry.Name()})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].version < files[j].version })
	latest := 0
	if len(files) > 0 {
		latest = files[len(files)-1].version
	}
	return files, latest, nil
}

func readUp(fsys fs.FS, name string) (string, error) {
	raw, err := fs.ReadFile(fsys, name)
	if err != nil {
		return "", fmt.Errorf("read migration %s: %w", name, err)
	}
	text := string(raw)
	upStart := strings.Index(text, "-- +goose Up")
	if upStart < 0 {
		return "", fmt.Errorf("migration %s has no Up section", name)
	}
	up := text[upStart+len("-- +goose Up"):]
	if downStart := strings.Index(up, "-- +goose Down"); downStart >= 0 {
		up = up[:downStart]
	}
	return strings.TrimSpace(up), nil
}

func pruneBackups(path string, keep int) error {
	matches, err := filepath.Glob(path + ".bak-*")
	if err != nil {
		return fmt.Errorf("list backups: %w", err)
	}
	sort.Strings(matches)
	for len(matches) > keep {
		if err := os.Remove(matches[0]); err != nil {
			return fmt.Errorf("remove old backup: %w", err)
		}
		matches = matches[1:]
	}
	return nil
}
