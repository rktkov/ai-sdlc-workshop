package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/library"
)

// LibraryRepository stores saved and downloaded videos.
type LibraryRepository struct {
	db *sql.DB
}

// NewLibraryRepository returns a repository over the given database.
func NewLibraryRepository(db *sql.DB) *LibraryRepository {
	return &LibraryRepository{db: db}
}

// Get returns one library entry.
func (r *LibraryRepository) Get(ctx context.Context, userID, videoID string) (library.Entry, error) {
	row := r.db.QueryRowContext(ctx, librarySelect+` WHERE user_id = ? AND video_id = ?`, userID, videoID)
	entry, err := scanLibrary(row)
	if errors.Is(err, sql.ErrNoRows) {
		return library.Entry{}, library.ErrNotFound
	}
	if err != nil {
		return library.Entry{}, err
	}
	return entry, nil
}

// List returns entries ordered by save time.
func (r *LibraryRepository) List(ctx context.Context, userID string) ([]library.Entry, error) {
	rows, err := r.db.QueryContext(ctx, librarySelect+` WHERE user_id = ? ORDER BY saved_at`, userID)
	if err != nil {
		return nil, fmt.Errorf("list library: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []library.Entry
	for rows.Next() {
		entry, err := scanLibrary(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate library: %w", err)
	}
	return out, nil
}

// Save inserts one saved entry. A duplicate returns ErrExists.
func (r *LibraryRepository) Save(ctx context.Context, entry library.Entry) error {
	result, err := r.db.ExecContext(ctx, `INSERT INTO library_entries
		(user_id, video_id, state, title, channel_id, duration_seconds,
		 file_extension, file_bytes, saved_at, downloaded_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, video_id) DO NOTHING`,
		entry.UserID, entry.VideoID, string(entry.State), entry.Title, entry.ChannelID,
		entry.DurationSeconds, entry.FileExtension, entry.FileBytes, entry.SavedAt,
		nullableTime(entry.DownloadedAt), entry.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert library entry: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check library insert: %w", err)
	}
	if rows == 0 {
		return library.ErrExists
	}
	return nil
}

// MarkDownloaded records the file metadata after a successful download.
func (r *LibraryRepository) MarkDownloaded(ctx context.Context, userID, videoID, extension string, bytes int64, now time.Time) (library.Entry, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE library_entries SET
		state = 'downloaded', file_extension = ?, file_bytes = ?, downloaded_at = ?, updated_at = ?
		WHERE user_id = ? AND video_id = ?`,
		extension, bytes, now, now, userID, videoID,
	)
	if err != nil {
		return library.Entry{}, fmt.Errorf("mark downloaded: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return library.Entry{}, fmt.Errorf("check mark downloaded: %w", err)
	}
	if rows == 0 {
		return library.Entry{}, library.ErrNotFound
	}
	return r.Get(ctx, userID, videoID)
}

// Remove deletes one entry.
func (r *LibraryRepository) Remove(ctx context.Context, userID, videoID string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM library_entries WHERE user_id = ? AND video_id = ?`, userID, videoID)
	if err != nil {
		return fmt.Errorf("remove library entry: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check library remove: %w", err)
	}
	if rows == 0 {
		return library.ErrNotFound
	}
	return nil
}

const librarySelect = `SELECT user_id, video_id, state, title, channel_id, duration_seconds,
	file_extension, file_bytes, saved_at, downloaded_at, updated_at FROM library_entries`

func scanLibrary(row scanner) (library.Entry, error) {
	var entry library.Entry
	var state string
	var downloaded sql.NullTime
	err := row.Scan(&entry.UserID, &entry.VideoID, &state, &entry.Title, &entry.ChannelID,
		&entry.DurationSeconds, &entry.FileExtension, &entry.FileBytes, &entry.SavedAt,
		&downloaded, &entry.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return library.Entry{}, err
		}
		return library.Entry{}, fmt.Errorf("scan library entry: %w", err)
	}
	entry.State = library.State(state)
	if downloaded.Valid {
		entry.DownloadedAt = downloaded.Time
	}
	return entry, nil
}

var _ library.Repository = (*LibraryRepository)(nil)
