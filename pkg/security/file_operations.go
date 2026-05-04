package security

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sync/singleflight"
)

var fileGroup singleflight.Group

func AtomicWriteJSON(data any, path string) error {

	_, err, _ := fileGroup.Do(path, func() (any, error) {
		return nil, atomicWrite(data, path)
	})

	if err != nil {
		return fmt.Errorf("singleflight atomic write: %w", err)
	}
	return nil
}

func atomicWrite(data any, path string) error {

	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	suffix, err := generateRandomSuffix()
	if err != nil {
		return err
	}
	tempPath := path + ".tmp." + suffix

	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tempPath)

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("encode data: %w", err)
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync file: %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}

	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("atomic rename: %w", err)
	}

	return nil
}

func generateRandomSuffix() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate random suffix: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func SafeReadJSON(path string, target any) error {

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return initializeEmptyFile(path, target)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	if len(data) == 0 {
		return initializeEmptyFile(path, target)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("unmarshal json: %w", err)
	}

	return nil
}

func initializeEmptyFile(path string, target any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	if err := AtomicWriteJSON(target, path); err != nil {
		return fmt.Errorf("initialize empty JSON file: %w", err)
	}
	return nil
}
