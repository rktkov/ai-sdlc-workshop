// Package player launches local media files.
package player

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Player opens one file with a configured or OS-default command.
type Player struct {
	Command string
	Args    []string
	GOOS    string
	Wait    bool
}

// New parses a PLAYER_CMD-style command.
func New(command string) *Player {
	fields := strings.Fields(strings.TrimSpace(command))
	if len(fields) == 0 {
		return &Player{}
	}
	return &Player{Command: fields[0], Args: fields[1:]}
}

// Open launches the player. By default it does not wait for the process.
func (p Player) Open(ctx context.Context, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("media file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("media file is not regular")
	}
	name, args, err := p.command(path)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, name, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start player: %w", err)
	}
	if p.Wait {
		return cmd.Wait()
	}
	return cmd.Process.Release()
}

func (p Player) command(path string) (string, []string, error) {
	if strings.TrimSpace(p.Command) != "" {
		args := append(append([]string(nil), p.Args...), path)
		return p.Command, args, nil
	}
	goos := p.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	switch goos {
	case "darwin":
		return "open", []string{path}, nil
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", path}, nil
	default:
		return "xdg-open", []string{path}, nil
	}
}
