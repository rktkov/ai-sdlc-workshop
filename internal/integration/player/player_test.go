package player

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rktkov/ytfeed/internal/testkit"
)

func TestOpenWithCommand(t *testing.T) {
	dir := t.TempDir()
	media := filepath.Join(dir, "video.mp4")
	if err := os.WriteFile(media, []byte("media"), 0o600); err != nil {
		t.Fatalf("write media: %v", err)
	}
	argsFile := filepath.Join(dir, "args.txt")
	script := testkit.WriteScript(t, dir, "player", "#!/bin/sh\nprintf '%s' \"$1\" > "+argsFile+"\n")
	p := &Player{Command: script, Wait: true}
	if err := p.Open(context.Background(), media); err != nil {
		t.Fatalf("Open: %v", err)
	}
	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	if string(data) != media {
		t.Fatalf("args = %q, want %q", data, media)
	}
}

func TestOpenMissingFile(t *testing.T) {
	if err := (&Player{Command: "/bin/true"}).Open(context.Background(), "/missing/video.mp4"); err == nil {
		t.Fatal("Open(missing) = nil, want an error")
	}
}

func TestOpenBadCommand(t *testing.T) {
	dir := t.TempDir()
	media := filepath.Join(dir, "video.mp4")
	if err := os.WriteFile(media, []byte("media"), 0o600); err != nil {
		t.Fatalf("write media: %v", err)
	}
	if err := (&Player{Command: filepath.Join(dir, "missing"), Wait: true}).Open(context.Background(), media); err == nil {
		t.Fatal("Open(bad command) = nil, want an error")
	}
}

func TestCommandDefault(t *testing.T) {
	tests := map[string]string{
		"darwin":  "open",
		"windows": "rundll32",
		"linux":   "xdg-open",
	}
	for goos, want := range tests {
		name, _, err := (Player{GOOS: goos}).command("/video.mp4")
		if err != nil {
			t.Fatalf("command(%s): %v", goos, err)
		}
		if name != want {
			t.Fatalf("command(%s) = %q, want %q", goos, name, want)
		}
	}
}

func TestNew(t *testing.T) {
	p := New("mpv --fs")
	if p.Command != "mpv" || len(p.Args) != 1 || p.Args[0] != "--fs" {
		t.Fatalf("player = %+v", p)
	}
	if New("").Command != "" {
		t.Fatal("New(empty) set a command")
	}
}
