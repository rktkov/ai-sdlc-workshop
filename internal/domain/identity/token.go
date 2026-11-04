package identity

import (
	"context"
	"errors"
)

var (
	// ErrTokenNotFound means no token file exists.
	ErrTokenNotFound = errors.New("token not found")
	// ErrInsecurePermissions means the token file is readable by other users.
	ErrInsecurePermissions = errors.New("token file permissions are too open")
)

// Store persists the credential bytes outside the database.
type Store interface {
	Read(ctx context.Context) (Token, error)
	Write(ctx context.Context, token Token) error
	Delete(ctx context.Context) error
	Path() string
}
