//go:build windows

package security

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

func WithFileLock(path string, fn func() error) error {
	lockPath := path + ".lock"
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return fmt.Errorf("create lock directory: %w", err)
	}
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open lock file: %w", err)
	}
	defer file.Close()

	overlapped := &windows.Overlapped{}
	handle := windows.Handle(file.Fd())
	if err := windows.LockFileEx(handle, windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, overlapped); err != nil {
		return fmt.Errorf("acquire file lock: %w", err)
	}
	defer windows.UnlockFileEx(handle, 0, 1, 0, overlapped)

	return fn()
}
