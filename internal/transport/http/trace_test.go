package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/rktkov/ytfeed/internal/pkg/trace"
)

func TestTraceMiddleware(t *testing.T) {
	dir := t.TempDir()
	logger := trace.New(dir)
	handler := TraceMiddleware(logger, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/feed", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d", recorder.Code)
	}

	entries, err := os.ReadDir(filepath.Join(dir, "traces"))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("trace files = %d", len(entries))
	}
	data, err := os.ReadFile(filepath.Join(dir, "traces", entries[0].Name()))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var span map[string]any
	if err := json.Unmarshal(data, &span); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if span["name"] != "http.request" {
		t.Fatalf("span = %+v", span)
	}
	attrs, _ := span["attributes"].(map[string]any)
	if attrs["path"] != "/v1/feed" || attrs["method"] != "GET" || attrs["status"] != float64(http.StatusNoContent) {
		t.Fatalf("attributes = %+v", attrs)
	}
}

func TestTraceMiddlewareNil(t *testing.T) {
	called := false
	handler := TraceMiddleware(nil, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if !called {
		t.Fatal("handler was not called")
	}
}

func TestTraceMiddlewareKeepsFlusher(t *testing.T) {
	logger := trace.New(t.TempDir())
	handler := TraceMiddleware(logger, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("traced writer does not implement http.Flusher")
		}
		_, _ = w.Write([]byte("event: card.upsert\ndata: {}\n\n"))
		flusher.Flush()
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/feed", nil))
	if recorder.Body.String() == "" {
		t.Fatal("SSE body is empty")
	}
}
