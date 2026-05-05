package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWithFileLockIgnoresStaleLockFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	lockPath := path + ".lock"
	if err := os.WriteFile(lockPath, []byte("stale"), 0o600); err != nil {
		t.Fatalf("write stale lock: %v", err)
	}

	called := false
	if err := WithFileLock(path, func() error {
		called = true
		return nil
	}); err != nil {
		t.Fatalf("with file lock: %v", err)
	}
	if !called {
		t.Fatal("lock callback was not called")
	}
}
