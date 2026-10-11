package storage

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestLocalFSSaveOpenDelete(t *testing.T) {
	ctx := context.Background()
	store, err := NewLocalFS(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	const key = "attachments/3f/9a1c"
	if err := store.Save(ctx, key, strings.NewReader("first")); err != nil {
		t.Fatalf("save: %v", err)
	}
	// Saving again under the same key replaces the file.
	if err := store.Save(ctx, key, strings.NewReader("second")); err != nil {
		t.Fatalf("save again: %v", err)
	}

	f, err := store.Open(ctx, key)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	body, _ := io.ReadAll(f)
	f.Close()
	if string(body) != "second" {
		t.Errorf("read %q, want %q", body, "second")
	}

	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.Open(ctx, key); !errors.Is(err, ErrNotFound) {
		t.Errorf("open after delete: %v, want ErrNotFound", err)
	}
	if err := store.Delete(ctx, key); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete twice: %v, want ErrNotFound", err)
	}
}

func TestLocalFSRefusesKeysOutsideTheStore(t *testing.T) {
	ctx := context.Background()
	store, err := NewLocalFS(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	for _, key := range []string{"", "../outside", "a/../../outside", "/etc/passwd"} {
		t.Run(key, func(t *testing.T) {
			if err := store.Save(ctx, key, strings.NewReader("x")); !errors.Is(err, ErrInvalidKey) {
				t.Errorf("save: %v, want ErrInvalidKey", err)
			}
			if _, err := store.Open(ctx, key); !errors.Is(err, ErrInvalidKey) {
				t.Errorf("open: %v, want ErrInvalidKey", err)
			}
			if err := store.Delete(ctx, key); !errors.Is(err, ErrInvalidKey) {
				t.Errorf("delete: %v, want ErrInvalidKey", err)
			}
		})
	}
}
