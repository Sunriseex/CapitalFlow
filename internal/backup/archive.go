package backup

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	CurrentFormatVersion = 1
	manifestName         = "manifest.json"
	dumpName             = "database.dump"
)

type Manifest struct {
	FormatVersion  int       `json:"format_version"`
	AppVersion     string    `json:"app_version"`
	SchemaVersion  int64     `json:"schema_version"`
	CreatedAt      time.Time `json:"created_at"`
	BaseCurrency   string    `json:"base_currency,omitempty"`
	DatabaseSHA256 string    `json:"database_sha256"`
}

func CreateArchive(ctx context.Context, path string, manifest *Manifest, dump io.Reader) (err error) {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("backup path is required")
	}
	if dump == nil {
		return fmt.Errorf("database dump is required")
	}
	if manifest == nil {
		return fmt.Errorf("backup manifest is required")
	}
	if manifest.FormatVersion != CurrentFormatVersion {
		return fmt.Errorf("backup format version must be %d", CurrentFormatVersion)
	}

	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return fmt.Errorf("create backup directory: %w", err)
	}
	dumpFile, err := os.CreateTemp(directory, ".capitalflow-dump-*")
	if err != nil {
		return fmt.Errorf("create temporary dump: %w", err)
	}
	dumpPath := dumpFile.Name()
	defer os.Remove(dumpPath)

	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(dumpFile, hash), &contextReader{ctx: ctx, reader: dump}); err != nil {
		_ = dumpFile.Close()
		return fmt.Errorf("copy database dump: %w", err)
	}
	if err := dumpFile.Close(); err != nil {
		return fmt.Errorf("close temporary dump: %w", err)
	}
	archiveManifest := *manifest
	archiveManifest.DatabaseSHA256 = hex.EncodeToString(hash.Sum(nil))

	archiveFile, err := os.CreateTemp(directory, ".capitalflow-backup-*")
	if err != nil {
		return fmt.Errorf("create temporary backup: %w", err)
	}
	archivePath := archiveFile.Name()
	defer func() {
		if err != nil {
			_ = os.Remove(archivePath)
		}
	}()

	zipWriter := zip.NewWriter(archiveFile)
	manifestEntry, err := zipWriter.Create(manifestName)
	if err != nil {
		return fmt.Errorf("create manifest entry: %w", err)
	}
	encoder := json.NewEncoder(manifestEntry)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(&archiveManifest); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	dumpEntry, err := zipWriter.Create(dumpName)
	if err != nil {
		return fmt.Errorf("create dump entry: %w", err)
	}
	storedDump, err := os.Open(dumpPath)
	if err != nil {
		return fmt.Errorf("open temporary dump: %w", err)
	}
	if _, err := io.Copy(dumpEntry, &contextReader{ctx: ctx, reader: storedDump}); err != nil {
		_ = storedDump.Close()
		return fmt.Errorf("write database dump: %w", err)
	}
	if err := storedDump.Close(); err != nil {
		return fmt.Errorf("close stored dump: %w", err)
	}
	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("close backup archive: %w", err)
	}
	if err := archiveFile.Close(); err != nil {
		return fmt.Errorf("close backup file: %w", err)
	}
	if err := os.Chmod(archivePath, 0o600); err != nil {
		return fmt.Errorf("secure backup file: %w", err)
	}
	if err := os.Rename(archivePath, path); err != nil {
		return fmt.Errorf("publish backup file: %w", err)
	}
	return nil
}

func ExtractArchive(ctx context.Context, path string, destination io.Writer) (Manifest, error) {
	if destination == nil {
		return Manifest{}, fmt.Errorf("dump destination is required")
	}
	archive, err := zip.OpenReader(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("open backup archive: %w", err)
	}
	defer archive.Close()

	manifestFile := findEntry(archive.File, manifestName)
	dumpFile := findEntry(archive.File, dumpName)
	if manifestFile == nil || dumpFile == nil {
		return Manifest{}, fmt.Errorf("backup must contain %s and %s", manifestName, dumpName)
	}
	manifestReader, err := manifestFile.Open()
	if err != nil {
		return Manifest{}, fmt.Errorf("open backup manifest: %w", err)
	}
	var manifest Manifest
	decodeErr := json.NewDecoder(manifestReader).Decode(&manifest)
	closeErr := manifestReader.Close()
	if decodeErr != nil {
		return Manifest{}, fmt.Errorf("decode backup manifest: %w", decodeErr)
	}
	if closeErr != nil {
		return Manifest{}, fmt.Errorf("close backup manifest: %w", closeErr)
	}
	if manifest.FormatVersion != CurrentFormatVersion {
		return Manifest{}, fmt.Errorf("unsupported backup format version %d", manifest.FormatVersion)
	}

	dumpReader, err := dumpFile.Open()
	if err != nil {
		return Manifest{}, fmt.Errorf("open database dump: %w", err)
	}
	hash := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(destination, hash), &contextReader{ctx: ctx, reader: dumpReader})
	closeErr = dumpReader.Close()
	if copyErr != nil {
		return Manifest{}, fmt.Errorf("extract database dump: %w", copyErr)
	}
	if closeErr != nil {
		return Manifest{}, fmt.Errorf("close database dump: %w", closeErr)
	}
	actualChecksum := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actualChecksum, manifest.DatabaseSHA256) {
		return Manifest{}, fmt.Errorf("database dump checksum mismatch")
	}
	return manifest, nil
}

func findEntry(files []*zip.File, name string) *zip.File {
	for _, file := range files {
		if file.Name == name {
			return file
		}
	}
	return nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(buffer []byte) (int, error) {
	select {
	case <-r.ctx.Done():
		return 0, fmt.Errorf("backup operation cancelled: %w", r.ctx.Err())
	default:
		count, err := r.reader.Read(buffer)
		if errors.Is(err, io.EOF) {
			return count, io.EOF
		}
		if err != nil {
			return count, fmt.Errorf("read backup data: %w", err)
		}
		return count, nil
	}
}
