package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/analysis"
)

func schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"topic":{"type":"string"}}}`)
}

func TestComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer key" {
			t.Errorf("Authorization = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		var request map[string]any
		if err := json.Unmarshal(body, &request); err != nil {
			t.Fatalf("request json: %v", err)
		}
		if request["model"] != "model-1" {
			t.Errorf("model = %v", request["model"])
		}
		messages, ok := request["messages"].([]any)
		if !ok || len(messages) != 2 {
			t.Errorf("messages = %v", request["messages"])
		}
		format, ok := request["response_format"].(map[string]any)
		if !ok || format["type"] != "json_schema" {
			t.Errorf("response_format = %v", request["response_format"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"choices":[{"message":{"content":"{\"topic\":\"ai-agents\"}"}}]}`)
	}))
	defer server.Close()

	client := New(server.URL+"/v1", "model-1", "key", true, time.Second)
	got, err := client.Complete(context.Background(), "system", "user", schema())
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if !strings.Contains(got, "ai-agents") {
		t.Fatalf("content = %q", got)
	}
}

func TestCompleteJSONObjectMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"type":"json_object"`) {
			t.Errorf("body = %s", body)
		}
		_, _ = fmt.Fprint(w, `{"choices":[{"message":{"content":"{}"}}]}`)
	}))
	defer server.Close()

	client := New(server.URL, "model", "", false, time.Second)
	if _, err := client.Complete(context.Background(), "s", "u", schema()); err != nil {
		t.Fatalf("Complete: %v", err)
	}
}

func TestCompleteTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = fmt.Fprint(w, `{"choices":[{"message":{"content":"{}"}}]}`)
	}))
	defer server.Close()

	client := New(server.URL, "model", "", true, 10*time.Millisecond)
	_, err := client.Complete(context.Background(), "s", "u", schema())
	if !errors.Is(err, analysis.ErrLLMTimeout) {
		t.Fatalf("Complete = %v, want ErrLLMTimeout", err)
	}
}

func TestCompleteUnhappy(t *testing.T) {
	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "boom", http.StatusInternalServerError)
		}))
		defer server.Close()
		if _, err := New(server.URL, "model", "", true, time.Second).Complete(context.Background(), "s", "u", schema()); err == nil {
			t.Fatal("Complete = nil, want an error")
		}
	})

	t.Run("malformed response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, `{`)
		}))
		defer server.Close()
		if _, err := New(server.URL, "model", "", true, time.Second).Complete(context.Background(), "s", "u", schema()); err == nil {
			t.Fatal("Complete = nil, want an error")
		}
	})

	t.Run("empty content", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, `{"choices":[{"message":{"content":""}}]}`)
		}))
		defer server.Close()
		if _, err := New(server.URL, "model", "", true, time.Second).Complete(context.Background(), "s", "u", schema()); err == nil {
			t.Fatal("Complete = nil, want an error")
		}
	})
}

func TestNewDefaults(t *testing.T) {
	client := New("", "", "", true, 0)
	if client.BaseURL != "http://localhost:11434/v1" || client.Model != "lfm2.5-1.2b-instruct" {
		t.Fatalf("client = %+v", client)
	}
	if client.Timeout != 60*time.Second {
		t.Fatalf("timeout = %v", client.Timeout)
	}
}
