// Package storage keeps files for the application: attachments, and whatever
// else a later phase stores (spec section 11). The application depends on the
// Store interface only, so files can move from the server's disk to another
// service without a change to the code that uses them.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// ErrNotFound is returned when no file is stored under a key.
var ErrNotFound = errors.New("file not found")

// ErrInvalidKey is returned for a key that is empty or points outside the store.
var ErrInvalidKey = errors.New("invalid storage key")

// Store keeps files under keys the application generates. A key is a relative
// path with forward slashes, such as "attachments/3f/9a1c". It is never a name
// a person supplied (spec section 11, generated storage identifiers).
type Store interface {
	Save(ctx context.Context, key string, r io.Reader) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

// LocalFS keeps files in a directory on the server's own disk.
type LocalFS struct {
	root string
}

func NewLocalFS(root string) (*LocalFS, error) {
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create storage directory %s: %w", root, err)
	}
	return &LocalFS{root: root}, nil
}

func (s *LocalFS) path(key string) (string, error) {
	if key == "" || !filepath.IsLocal(filepath.FromSlash(key)) {
		return "", fmt.Errorf("%w: %q", ErrInvalidKey, key)
	}
	return filepath.Join(s.root, filepath.FromSlash(key)), nil
}

// Save writes the file whole or not at all: it is written beside its final
// place under a temporary name and moved there once complete, so a reader
// never sees a partial file.
func (s *LocalFS) Save(_ context.Context, key string, r io.Reader) error {
	path, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", key, err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".saving-*")
	if err != nil {
		return fmt.Errorf("failed to save %s: %w", key, err)
	}
	defer os.Remove(tmp.Name())

	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to save %s: %w", key, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to save %s: %w", key, err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("failed to save %s: %w", key, err)
	}
	return nil
}

func (s *LocalFS) Open(_ context.Context, key string) (io.ReadCloser, error) {
	path, err := s.path(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", key, err)
	}
	return f, nil
}

func (s *LocalFS) Delete(_ context.Context, key string) error {
	path, err := s.path(key)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if errors.Is(err, fs.ErrNotExist) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to delete %s: %w", key, err)
	}
	return nil
}
