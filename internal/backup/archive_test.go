package backup

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestArchiveRoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "capitalflow.zip")
	createdAt := time.Date(2026, time.July, 8, 5, 0, 0, 0, time.UTC)
	want := Manifest{
		FormatVersion: 1,
		AppVersion:    "v0.6.2",
		SchemaVersion: 29,
		CreatedAt:     createdAt,
		BaseCurrency:  "RUB",
	}
	if err := CreateArchive(context.Background(), path, &want, strings.NewReader("postgres dump")); err != nil {
		t.Fatalf("create archive: %v", err)
	}

	var dump bytes.Buffer
	got, err := ExtractArchive(context.Background(), path, &dump)
	if err != nil {
		t.Fatalf("extract archive: %v", err)
	}
	if got.FormatVersion != want.FormatVersion || got.AppVersion != want.AppVersion ||
		got.SchemaVersion != want.SchemaVersion || !got.CreatedAt.Equal(want.CreatedAt) ||
		got.BaseCurrency != want.BaseCurrency {
		t.Fatalf("manifest = %#v, want %#v", got, want)
	}
	if dump.String() != "postgres dump" {
		t.Fatalf("dump = %q", dump.String())
	}
}

func TestExtractArchiveRejectsCorruptDump(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "capitalflow.zip")
	manifest := Manifest{FormatVersion: 1}
	if err := CreateArchive(context.Background(), path, &manifest, strings.NewReader("original")); err != nil {
		t.Fatalf("create archive: %v", err)
	}

	rewriteDump(t, path, "corrupt")

	_, err := ExtractArchive(context.Background(), path, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("error = %v, want checksum failure", err)
	}
}

func rewriteDump(t *testing.T, path, replacement string) {
	t.Helper()

	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	manifest, err := reader.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	manifestBytes := new(bytes.Buffer)
	if _, err := manifestBytes.ReadFrom(manifest); err != nil {
		t.Fatal(err)
	}
	if err := manifest.Close(); err != nil {
		t.Fatal(err)
	}

	temporary := path + ".tmp"
	file, err := os.Create(temporary)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	manifestFile, _ := writer.Create("manifest.json")
	_, _ = manifestFile.Write(manifestBytes.Bytes())
	dumpFile, _ := writer.Create("database.dump")
	_, _ = dumpFile.Write([]byte(replacement))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(temporary, path); err != nil {
		t.Fatal(err)
	}
}
