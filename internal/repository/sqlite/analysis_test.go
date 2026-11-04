package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/analysis"
	"github.com/rktkov/ytfeed/internal/storage"
)

func analyzedFixture() analysis.Analysis {
	now := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	return analysis.Analysis{
		ID: "a1", UserID: "owner", VideoID: "dQw4w9WgXcQ",
		ProfileVersion: 1, RubricVersion: "v1", ModelID: "model-1",
		State: analysis.StateAnalyzed, Topic: "ai-agents", TLDR: "tldr",
		AxisScores: []analysis.AxisScore{
			{Axis: "depth", Score: 0.8, Justification: "good"},
			{Axis: "novelty", Score: 0.5, Justification: "ok"},
		},
		MarketingScore: 0.1, AISlopScore: 0.2,
		Similarity: "new territory", NewTerritory: true, WhyThisCard: "why",
		TranscriptHash: "abc", TranscriptBytes: 10,
		CreatedAt: now, CompletedAt: now,
	}
}

func TestAnalysisSaveGet(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewAnalysisRepository(db)
	want := analyzedFixture()
	if err := repo.Save(context.Background(), want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := repo.Get(context.Background(), "a1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Topic != want.Topic || got.ModelID != want.ModelID || got.State != analysis.StateAnalyzed {
		t.Fatalf("got = %+v", got)
	}
	if len(got.AxisScores) != 2 || got.AxisScores[0].Axis != "depth" {
		t.Fatalf("scores = %+v", got.AxisScores)
	}

	found, err := repo.FindByKey(context.Background(), "owner", "dQw4w9WgXcQ", "v1")
	if err != nil {
		t.Fatalf("FindByKey: %v", err)
	}
	if found.ID != "a1" {
		t.Fatalf("found = %+v", found)
	}
}

func TestAnalysisFailed(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	now := time.Now().UTC()
	failed := analysis.Analysis{
		ID: "a2", UserID: "owner", VideoID: "dQw4w9WgXcQ",
		ProfileVersion: 1, RubricVersion: "v1", ModelID: "model-1",
		State: analysis.StateFailed, Reason: analysis.ReasonNoTranscript,
		CreatedAt: now, CompletedAt: now,
	}
	repo := NewAnalysisRepository(db)
	if err := repo.Save(context.Background(), failed); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := repo.FindByKey(context.Background(), "owner", "dQw4w9WgXcQ", "v1")
	if err != nil {
		t.Fatalf("FindByKey: %v", err)
	}
	if got.State != analysis.StateFailed || got.Reason != analysis.ReasonNoTranscript || len(got.AxisScores) != 0 {
		t.Fatalf("got = %+v", got)
	}
}

func TestAnalysisDuplicate(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewAnalysisRepository(db)
	if err := repo.Save(context.Background(), analyzedFixture()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	duplicate := analyzedFixture()
	duplicate.ID = "a2"
	if err := repo.Save(context.Background(), duplicate); !errors.Is(err, analysis.ErrAnalysisExists) {
		t.Fatalf("Save duplicate = %v, want ErrAnalysisExists", err)
	}
}

func TestAnalysisNotFound(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewAnalysisRepository(db)
	if _, err := repo.Get(context.Background(), "missing"); !errors.Is(err, analysis.ErrAnalysisNotFound) {
		t.Fatalf("Get = %v, want ErrAnalysisNotFound", err)
	}
	if _, err := repo.FindByKey(context.Background(), "owner", "dQw4w9WgXcQ", "v1"); !errors.Is(err, analysis.ErrAnalysisNotFound) {
		t.Fatalf("FindByKey = %v, want ErrAnalysisNotFound", err)
	}
}

func TestAnalysisInvalid(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	invalid := analyzedFixture()
	invalid.State = "pending"
	if err := NewAnalysisRepository(db).Save(context.Background(), invalid); err == nil {
		t.Fatal("Save(invalid) = nil, want an error")
	}
}

func TestAnalysisListForVideo(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewAnalysisRepository(db)
	first := analyzedFixture()
	if err := repo.Save(context.Background(), first); err != nil {
		t.Fatalf("Save first: %v", err)
	}
	second := analyzedFixture()
	second.ID = "a2"
	second.RubricVersion = "v2"
	second.CompletedAt = second.CompletedAt.Add(time.Hour)
	if err := repo.Save(context.Background(), second); err != nil {
		t.Fatalf("Save second: %v", err)
	}
	list, err := repo.ListForVideo(context.Background(), "dQw4w9WgXcQ")
	if err != nil {
		t.Fatalf("ListForVideo: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("list = %d, want 2", len(list))
	}
}

func TestAnalysisStoreError(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec("DROP TABLE analysis_axis_scores"); err != nil {
		t.Fatalf("drop scores: %v", err)
	}
	if err := NewAnalysisRepository(db).Save(context.Background(), analyzedFixture()); err == nil {
		t.Fatal("Save = nil, want an error")
	}
}
