package backup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CreateOptions struct {
	DatabaseURL string
	OutputPath  string
	AppVersion  string
}

type RestoreOptions struct {
	DatabaseURL string
	InputPath   string
}

func Create(ctx context.Context, options CreateOptions) (Manifest, error) {
	if strings.TrimSpace(options.DatabaseURL) == "" {
		return Manifest{}, fmt.Errorf("database url is required")
	}
	if strings.TrimSpace(options.OutputPath) == "" {
		return Manifest{}, fmt.Errorf("backup output path is required")
	}

	pool, err := pgxpool.New(ctx, options.DatabaseURL)
	if err != nil {
		return Manifest{}, fmt.Errorf("connect for backup metadata: %w", err)
	}
	defer pool.Close()

	manifest := Manifest{
		FormatVersion: CurrentFormatVersion,
		AppVersion:    strings.TrimSpace(options.AppVersion),
		CreatedAt:     time.Now().UTC(),
	}
	if err := pool.QueryRow(ctx, `SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version WHERE is_applied`).Scan(&manifest.SchemaVersion); err != nil {
		return Manifest{}, fmt.Errorf("read schema version: %w", err)
	}
	if err := pool.QueryRow(ctx, `SELECT primary_currency FROM users ORDER BY created_at, id LIMIT 1`).Scan(&manifest.BaseCurrency); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Manifest{}, fmt.Errorf("read base currency: %w", err)
	}
	manifest.BaseCurrency = strings.TrimSpace(manifest.BaseCurrency)

	temporary, err := os.CreateTemp("", "capitalflow-pg-dump-*")
	if err != nil {
		return Manifest{}, fmt.Errorf("create temporary database dump: %w", err)
	}
	dumpPath := temporary.Name()
	if err := temporary.Close(); err != nil {
		return Manifest{}, fmt.Errorf("close temporary database dump: %w", err)
	}
	defer os.Remove(dumpPath)

	if output, err := exec.CommandContext(
		ctx, "pg_dump",
		"--format=custom",
		"--no-owner",
		"--no-privileges",
		"--file", dumpPath,
		options.DatabaseURL,
	).CombinedOutput(); err != nil {
		return Manifest{}, commandError("pg_dump", output, err)
	}
	dump, err := os.Open(dumpPath)
	if err != nil {
		return Manifest{}, fmt.Errorf("open database dump: %w", err)
	}
	defer dump.Close()
	if err := CreateArchive(ctx, options.OutputPath, &manifest, dump); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func Restore(ctx context.Context, options RestoreOptions) (Manifest, error) {
	if strings.TrimSpace(options.DatabaseURL) == "" {
		return Manifest{}, fmt.Errorf("database url is required")
	}
	if strings.TrimSpace(options.InputPath) == "" {
		return Manifest{}, fmt.Errorf("backup input path is required")
	}

	pool, err := pgxpool.New(ctx, options.DatabaseURL)
	if err != nil {
		return Manifest{}, fmt.Errorf("connect to restore target: %w", err)
	}
	var relationCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM pg_class AS relation
		JOIN pg_namespace AS namespace ON namespace.oid = relation.relnamespace
		WHERE namespace.nspname = 'public'
		  AND relation.relkind IN ('r', 'p', 'v', 'm', 'S')
	`).Scan(&relationCount); err != nil {
		pool.Close()
		return Manifest{}, fmt.Errorf("inspect restore target: %w", err)
	}
	pool.Close()
	if relationCount != 0 {
		return Manifest{}, fmt.Errorf("restore target must be an empty database")
	}

	temporary, err := os.CreateTemp("", "capitalflow-restore-*")
	if err != nil {
		return Manifest{}, fmt.Errorf("create temporary restore dump: %w", err)
	}
	dumpPath := temporary.Name()
	defer os.Remove(dumpPath)
	manifest, err := ExtractArchive(ctx, options.InputPath, temporary)
	if err != nil {
		_ = temporary.Close()
		return Manifest{}, err
	}
	if err := temporary.Close(); err != nil {
		return Manifest{}, fmt.Errorf("close temporary restore dump: %w", err)
	}

	// #nosec G204 -- the database URL and verified temporary dump are explicit CLI inputs to pg_restore.
	if output, err := exec.CommandContext(
		ctx, "pg_restore",
		"--exit-on-error",
		"--no-owner",
		"--no-privileges",
		"--dbname", options.DatabaseURL,
		dumpPath,
	).CombinedOutput(); err != nil {
		return Manifest{}, commandError("pg_restore", output, err)
	}

	verificationPool, err := pgxpool.New(ctx, options.DatabaseURL)
	if err != nil {
		return Manifest{}, fmt.Errorf("connect to restored database: %w", err)
	}
	defer verificationPool.Close()
	var schemaVersion int64
	if err := verificationPool.QueryRow(ctx, `SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version WHERE is_applied`).Scan(&schemaVersion); err != nil {
		return Manifest{}, fmt.Errorf("verify restored schema version: %w", err)
	}
	if schemaVersion != manifest.SchemaVersion {
		return Manifest{}, fmt.Errorf("restored schema version %d does not match backup %d", schemaVersion, manifest.SchemaVersion)
	}
	return manifest, nil
}

func commandError(name string, output []byte, err error) error {
	message := strings.TrimSpace(string(output))
	if message == "" {
		return fmt.Errorf("%s: %w", name, err)
	}
	return fmt.Errorf("%s: %s: %w", name, message, err)
}
