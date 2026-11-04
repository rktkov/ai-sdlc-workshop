package httpapi

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrCreateTokenUnhappy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "api-token")
	if err := os.WriteFile(path, []byte(""), 0o600); err != nil {
		t.Fatalf("write empty token: %v", err)
	}

	if _, err := LoadOrCreateToken(dir); err == nil {
		t.Fatal("LoadOrCreateToken(empty) = nil, want an error")
	}
}
