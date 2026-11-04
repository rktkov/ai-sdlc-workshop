package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/rktkov/ytfeed/internal/domain/taste"
	"github.com/rktkov/ytfeed/internal/storage"
)

func TestProfileRoundTrip(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewProfileRepository(db)
	want := taste.Profile{
		UserID:        "owner",
		Note:          "prefer depth",
		RubricVersion: "v1",
		Axes: []taste.Axis{
			{Name: "depth", Weight: 0.7, PositiveExamples: []string{"long"}, NegativeExamples: []string{"shallow"}},
			{Name: "novelty", Weight: 0.5, PositiveExamples: []string{"new"}, NegativeExamples: []string{"repeat"}},
			{Name: "clarity", Weight: 0.6, PositiveExamples: []string{"clear"}, NegativeExamples: []string{"vague"}},
		},
		Interests: []string{"ai-agents", "local-models"},
	}
	if err := repo.Save(context.Background(), want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.Get(context.Background(), "owner")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Note != want.Note || got.RubricVersion != want.RubricVersion {
		t.Fatalf("profile = %+v", got)
	}
	if len(got.Axes) != len(want.Axes) {
		t.Fatalf("axes = %d, want %d", len(got.Axes), len(want.Axes))
	}
	for i, axis := range got.Axes {
		if axis.Name != want.Axes[i].Name || axis.Weight != want.Axes[i].Weight {
			t.Fatalf("axis %d = %+v, want %+v", i, axis, want.Axes[i])
		}
		if len(axis.PositiveExamples) != 1 || len(axis.NegativeExamples) != 1 {
			t.Fatalf("axis %d examples = %+v", i, axis)
		}
	}
	if len(got.Interests) != 2 || got.Interests[0] != "ai-agents" {
		t.Fatalf("interests = %v", got.Interests)
	}
}

func TestProfileUnhappy(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewProfileRepository(db)
	if _, err := repo.Get(context.Background(), "missing"); !errors.Is(err, taste.ErrProfileNotFound) {
		t.Fatalf("Get(missing) = %v, want ErrProfileNotFound", err)
	}

	invalid := taste.Profile{UserID: "owner"}
	if err := repo.Save(context.Background(), invalid); err == nil {
		t.Fatal("Save(invalid) = nil, want an error")
	}

	valid := taste.Profile{
		UserID:        "owner",
		RubricVersion: "v1",
		Axes: []taste.Axis{
			{Name: "a", Weight: 0.5, PositiveExamples: []string{"p"}, NegativeExamples: []string{"n"}},
			{Name: "b", Weight: 0.5, PositiveExamples: []string{"p"}, NegativeExamples: []string{"n"}},
			{Name: "c", Weight: 0.5, PositiveExamples: []string{"p"}, NegativeExamples: []string{"n"}},
		},
		Interests: []string{"x"},
	}
	if err := repo.Save(context.Background(), valid); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := repo.Save(context.Background(), valid); err == nil {
		t.Fatal("Save(duplicate) = nil, want an error")
	}
}

func TestProfileStoreErrors(t *testing.T) {
	valid := taste.Profile{
		UserID:        "owner",
		RubricVersion: "v1",
		Axes: []taste.Axis{
			{Name: "a", Weight: 0.5, PositiveExamples: []string{"p"}, NegativeExamples: []string{"n"}},
			{Name: "b", Weight: 0.5, PositiveExamples: []string{"p"}, NegativeExamples: []string{"n"}},
			{Name: "c", Weight: 0.5, PositiveExamples: []string{"p"}, NegativeExamples: []string{"n"}},
		},
		Interests: []string{"x"},
	}

	t.Run("missing examples table", func(t *testing.T) {
		db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer func() { _ = db.Close() }()
		repo := NewProfileRepository(db)
		if err := repo.Save(context.Background(), valid); err != nil {
			t.Fatalf("Save: %v", err)
		}
		if _, err := db.Exec("DROP TABLE taste_examples"); err != nil {
			t.Fatalf("drop examples: %v", err)
		}
		if _, err := repo.Get(context.Background(), valid.UserID); err == nil {
			t.Fatal("Get(missing examples) = nil, want an error")
		}
	})

	t.Run("missing axes table", func(t *testing.T) {
		db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer func() { _ = db.Close() }()
		repo := NewProfileRepository(db)
		if err := repo.Save(context.Background(), valid); err != nil {
			t.Fatalf("Save: %v", err)
		}
		if _, err := db.Exec("DROP TABLE taste_axes"); err != nil {
			t.Fatalf("drop axes: %v", err)
		}
		if _, err := repo.Get(context.Background(), valid.UserID); err == nil {
			t.Fatal("Get(missing axes) = nil, want an error")
		}
	})
}

func TestProfileSaveErrors(t *testing.T) {
	valid := taste.Profile{
		UserID:        "owner",
		RubricVersion: "v1",
		Axes: []taste.Axis{
			{Name: "a", Weight: 0.5, PositiveExamples: []string{"p"}, NegativeExamples: []string{"n"}},
			{Name: "b", Weight: 0.5, PositiveExamples: []string{"p"}, NegativeExamples: []string{"n"}},
			{Name: "c", Weight: 0.5, PositiveExamples: []string{"p"}, NegativeExamples: []string{"n"}},
		},
		Interests: []string{"x"},
	}

	for _, table := range []string{"taste_profiles", "taste_axes", "taste_examples", "taste_interests"} {
		t.Run(table, func(t *testing.T) {
			db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			defer func() { _ = db.Close() }()
			if _, err := db.Exec("DROP TABLE " + table); err != nil {
				t.Fatalf("drop %s: %v", table, err)
			}
			if err := NewProfileRepository(db).Save(context.Background(), valid); err == nil {
				t.Fatalf("Save(missing %s) = nil, want an error", table)
			}
		})
	}
}
