package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/rktkov/ytfeed/internal/domain/analysis"
)

// AnalysisRepository stores terminal analyses.
type AnalysisRepository struct {
	db *sql.DB
}

// NewAnalysisRepository returns a repository over the given database.
func NewAnalysisRepository(db *sql.DB) *AnalysisRepository {
	return &AnalysisRepository{db: db}
}

// Save inserts one terminal analysis and its axis scores.
func (r *AnalysisRepository) Save(ctx context.Context, value analysis.Analysis) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if value.ID == "" {
		return errors.New("analysis has no id")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin analysis: %w", err)
	}
	var exists int
	err = tx.QueryRowContext(ctx, `SELECT 1 FROM analyses
		WHERE user_id = ? AND video_id = ? AND rubric_version = ?`,
		value.UserID, value.VideoID, value.RubricVersion,
	).Scan(&exists)
	if err == nil {
		_ = tx.Rollback()
		return analysis.ErrAnalysisExists
	}
	if !errors.Is(err, sql.ErrNoRows) {
		_ = tx.Rollback()
		return fmt.Errorf("find analysis: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO analyses
		(analysis_id, user_id, video_id, profile_version, rubric_version, model_id, state, reason,
		 topic, tldr, marketing_score, ai_slop_score, similarity, new_territory, why_this_card,
		 transcript_sha256, transcript_bytes, created_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		value.ID, value.UserID, value.VideoID, value.ProfileVersion, value.RubricVersion,
		value.ModelID, string(value.State), string(value.Reason), value.Topic, value.TLDR,
		value.MarketingScore, value.AISlopScore, value.Similarity, boolInt(value.NewTerritory),
		value.WhyThisCard, value.TranscriptHash, value.TranscriptBytes, value.CreatedAt, value.CompletedAt,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("insert analysis: %w", err)
	}
	for position, score := range value.AxisScores {
		if _, err := tx.ExecContext(ctx, `INSERT INTO analysis_axis_scores
			(analysis_id, axis_name, score, justification, position) VALUES (?, ?, ?, ?, ?)`,
			value.ID, score.Axis, score.Score, score.Justification, position,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert axis score %s: %w", score.Axis, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit analysis: %w", err)
	}
	return nil
}

// Get returns one analysis by ID.
func (r *AnalysisRepository) Get(ctx context.Context, analysisID string) (analysis.Analysis, error) {
	row := r.db.QueryRowContext(ctx, analysisSelect+` WHERE analysis_id = ?`, analysisID)
	value, err := scanAnalysis(row)
	if errors.Is(err, sql.ErrNoRows) {
		return analysis.Analysis{}, analysis.ErrAnalysisNotFound
	}
	if err != nil {
		return analysis.Analysis{}, err
	}
	return r.withScores(ctx, value)
}

// FindByKey returns the terminal analysis for a natural key.
func (r *AnalysisRepository) FindByKey(ctx context.Context, userID, videoID, rubricVersion string) (analysis.Analysis, error) {
	row := r.db.QueryRowContext(ctx, analysisSelect+` WHERE user_id = ? AND video_id = ? AND rubric_version = ?`,
		userID, videoID, rubricVersion)
	value, err := scanAnalysis(row)
	if errors.Is(err, sql.ErrNoRows) {
		return analysis.Analysis{}, analysis.ErrAnalysisNotFound
	}
	if err != nil {
		return analysis.Analysis{}, err
	}
	return r.withScores(ctx, value)
}

// ListForVideo returns all analyses for a video, newest first.
func (r *AnalysisRepository) ListForVideo(ctx context.Context, videoID string) ([]analysis.Analysis, error) {
	rows, err := r.db.QueryContext(ctx, analysisSelect+` WHERE video_id = ? ORDER BY completed_at DESC`, videoID)
	if err != nil {
		return nil, fmt.Errorf("list analyses: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []analysis.Analysis
	for rows.Next() {
		value, err := scanAnalysis(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate analyses: %w", err)
	}
	for i := range out {
		value, err := r.withScores(ctx, out[i])
		if err != nil {
			return nil, err
		}
		out[i] = value
	}
	return out, nil
}

const analysisSelect = `SELECT analysis_id, user_id, video_id, profile_version, rubric_version,
	model_id, state, reason, topic, tldr, marketing_score, ai_slop_score, similarity,
	new_territory, why_this_card, transcript_sha256, transcript_bytes, created_at, completed_at
	FROM analyses`

type scanner interface {
	Scan(dest ...any) error
}

func scanAnalysis(row scanner) (analysis.Analysis, error) {
	var value analysis.Analysis
	var state, reason string
	var newTerritory int
	err := row.Scan(&value.ID, &value.UserID, &value.VideoID, &value.ProfileVersion,
		&value.RubricVersion, &value.ModelID, &state, &reason, &value.Topic, &value.TLDR,
		&value.MarketingScore, &value.AISlopScore, &value.Similarity, &newTerritory,
		&value.WhyThisCard, &value.TranscriptHash, &value.TranscriptBytes,
		&value.CreatedAt, &value.CompletedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return analysis.Analysis{}, err
		}
		return analysis.Analysis{}, fmt.Errorf("scan analysis: %w", err)
	}
	value.State = analysis.State(state)
	value.Reason = analysis.Reason(reason)
	value.NewTerritory = newTerritory != 0
	return value, nil
}

func (r *AnalysisRepository) withScores(ctx context.Context, value analysis.Analysis) (analysis.Analysis, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT axis_name, score, justification
		FROM analysis_axis_scores WHERE analysis_id = ? ORDER BY position`, value.ID)
	if err != nil {
		return analysis.Analysis{}, fmt.Errorf("read axis scores: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var score analysis.AxisScore
		if err := rows.Scan(&score.Axis, &score.Score, &score.Justification); err != nil {
			return analysis.Analysis{}, fmt.Errorf("scan axis score: %w", err)
		}
		value.AxisScores = append(value.AxisScores, score)
	}
	if err := rows.Err(); err != nil {
		return analysis.Analysis{}, fmt.Errorf("iterate axis scores: %w", err)
	}
	return value, nil
}
