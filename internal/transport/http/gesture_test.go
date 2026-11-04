package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/rktkov/ytfeed/internal/domain/taste"
	"github.com/rktkov/ytfeed/internal/repository/sqlite"
	"github.com/rktkov/ytfeed/internal/storage"
)

func TestGestureEndToEnd(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	service := &taste.Service{Store: sqlite.NewTasteRepository(db)}
	handler := HandlerWithGesture("secret", "dev", 2, GestureHandler(service, "owner"))

	body := []byte(`{
		"gesture_id": "g-1",
		"card_id": "card-1",
		"video_id": "video-1",
		"channel_id": "channel-1",
		"topic": "ai-agents",
		"gesture": "left",
		"reason": "marketing"
	}`)
	rec := request(t, handler, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}

	var response gestureResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Applied || response.Duplicate {
		t.Fatalf("response = %+v", response)
	}

	var signals, colds int
	if err := db.QueryRow("SELECT count(*) FROM signals").Scan(&signals); err != nil {
		t.Fatalf("count signals: %v", err)
	}
	if err := db.QueryRow("SELECT count(*) FROM cold_periods").Scan(&colds); err != nil {
		t.Fatalf("count cold periods: %v", err)
	}
	if signals != 1 || colds != 1 {
		t.Fatalf("signals = %d, cold periods = %d", signals, colds)
	}

	again := request(t, handler, body)
	if again.Code != http.StatusOK {
		t.Fatalf("duplicate status = %d: %s", again.Code, again.Body.String())
	}
	if err := db.QueryRow("SELECT count(*) FROM signals").Scan(&signals); err != nil {
		t.Fatalf("count signals: %v", err)
	}
	if signals != 1 {
		t.Fatalf("signals after retry = %d, want 1", signals)
	}
}

func TestGestureUnhappy(t *testing.T) {
	service := &taste.Service{Store: &stubStore{}}
	handler := HandlerWithGesture("secret", "dev", 2, GestureHandler(service, "owner"))

	t.Run("bad json", func(t *testing.T) {
		rec := request(t, handler, []byte(`{`))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("invalid gesture", func(t *testing.T) {
		body := []byte(`{"gesture_id":"g","card_id":"c","video_id":"v","channel_id":"ch","topic":"t","gesture":"up"}`)
		rec := request(t, handler, body)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})
}

type stubStore struct{}

func (s *stubStore) RecordGesture(context.Context, taste.Signal, *taste.ColdPeriod) (bool, error) {
	return true, nil
}

func request(t *testing.T, handler http.Handler, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/gestures", bytes.NewReader(body))
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}
