// Package tokenfile stores the OAuth token in one 0600 file.
package tokenfile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rktkov/ytfeed/internal/domain/identity"
)

// FileStore stores the token at <dataDir>/token.
type FileStore struct {
	path string
}

// New returns a token store under dataDir.
func New(dataDir string) *FileStore {
	return &FileStore{path: filepath.Join(dataDir, "token")}
}

// Path returns the token file path.
func (s *FileStore) Path() string {
	return s.path
}

// Read returns the stored token.
func (s *FileStore) Read(context.Context) (identity.Token, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return identity.Token{}, identity.ErrTokenNotFound
	}
	if err != nil {
		return identity.Token{}, fmt.Errorf("read token: %w", err)
	}
	var token identity.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return identity.Token{}, fmt.Errorf("decode token: %w", err)
	}
	if token.AccessToken == "" {
		return identity.Token{}, errors.New("token file has no access token")
	}
	return token, nil
}

// Write replaces the token atomically with 0600 permissions.
func (s *FileStore) Write(_ context.Context, token identity.Token) error {
	if token.AccessToken == "" {
		return errors.New("token has no access token")
	}
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("encode token: %w", err)
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}

	temp, err := os.CreateTemp(dir, ".token-*")
	if err != nil {
		return fmt.Errorf("create token file: %w", err)
	}
	tempPath := temp.Name()
	defer func() { _ = os.Remove(tempPath) }()

	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return fmt.Errorf("set token permissions: %w", err)
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write token: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close token: %w", err)
	}
	if err := os.Rename(tempPath, s.path); err != nil {
		return fmt.Errorf("replace token: %w", err)
	}
	return nil
}

// Delete removes the token file.
func (s *FileStore) Delete(context.Context) error {
	err := os.Remove(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("delete token: %w", err)
	}
	return nil
}

// CheckPermissions reports a token file that other users can read.
func (s *FileStore) CheckPermissions() error {
	info, err := os.Stat(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return identity.ErrTokenNotFound
		}
		return fmt.Errorf("stat token: %w", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return identity.ErrInsecurePermissions
	}
	return nil
}
