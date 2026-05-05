//go:build windows

package security

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func WithFileLock(path string, fn func() error) error {
	lockPath := path + ".lock"
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return fmt.Errorf("create lock directory: %w", err)
	}
	var file *os.File
	var err error

	for range 100 {
		file, err = os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
		if err == nil {
			defer os.Remove(lockPath)
			defer file.Close()
			return fn()
		}
		if !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("create lock file: %w", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	return fmt.Errorf("acquire file lock %s: %w", lockPath, err)
}
