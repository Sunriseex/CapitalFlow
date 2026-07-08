package backup

import (
	"context"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCreateAndRestoreIntoEmptyDatabase(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	if _, err := execLookPath("pg_dump"); err != nil {
		t.Skip("pg_dump is not installed")
	}
	if _, err := execLookPath("pg_restore"); err != nil {
		t.Skip("pg_restore is not installed")
	}

	ctx := t.Context()
	source, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect source database: %v", err)
	}
	defer source.Close()

	userID := uuid.NewString()
	accountID := uuid.NewString()
	transactionID := uuid.NewString()
	email := "backup-" + userID + "@example.test"
	now := time.Now().UTC().Truncate(time.Microsecond)
	if _, err := source.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, primary_currency, created_at, updated_at)
		VALUES ($1, $2, 'backup-test-hash', 'RUB', $3, $3)
	`, userID, email, now); err != nil {
		t.Fatalf("seed source user: %v", err)
	}
	if _, err := source.Exec(ctx, `
		INSERT INTO accounts (id, owner_user_id, name, bank, type, currency, is_active, opened_at, created_at, updated_at)
		VALUES ($1, $2, 'Backup Account', 'Test Bank', 'card', 'RUB', true, $3, $3, $3)
	`, accountID, userID, now); err != nil {
		t.Fatalf("seed source account: %v", err)
	}
	if _, err := source.Exec(ctx, `
		INSERT INTO transactions (id, account_id, type, amount, description, occurred_at, created_at)
		VALUES ($1, $2, 'income', 1234.56, 'Backup sentinel', $3, $3)
	`, transactionID, accountID, now); err != nil {
		t.Fatalf("seed source transaction: %v", err)
	}
	t.Cleanup(func() {
		_, _ = source.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	targetName := "capitalflow_restore_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	adminURL := databaseURLWithName(t, databaseURL, "postgres")
	targetURL := databaseURLWithName(t, databaseURL, targetName)
	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatalf("connect postgres admin database: %v", err)
	}
	defer admin.Close(ctx)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{targetName}.Sanitize()); err != nil {
		t.Fatalf("create restore target: %v", err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "DROP DATABASE IF EXISTS "+pgx.Identifier{targetName}.Sanitize()+" WITH (FORCE)")
	})

	archivePath := filepath.Join(t.TempDir(), "backup.zip")
	created, err := Create(ctx, CreateOptions{
		DatabaseURL: databaseURL,
		OutputPath:  archivePath,
		AppVersion:  "integration-test",
	})
	if err != nil {
		t.Fatalf("create backup: %v", err)
	}
	if _, err := Restore(ctx, RestoreOptions{DatabaseURL: databaseURL, InputPath: archivePath}); err == nil || !strings.Contains(err.Error(), "empty database") {
		t.Fatalf("restore into non-empty database error = %v", err)
	}
	restored, err := Restore(ctx, RestoreOptions{DatabaseURL: targetURL, InputPath: archivePath})
	if err != nil {
		t.Fatalf("restore backup: %v", err)
	}
	if restored.SchemaVersion != created.SchemaVersion {
		t.Fatalf("restored schema = %d, want %d", restored.SchemaVersion, created.SchemaVersion)
	}

	target, err := pgxpool.New(ctx, targetURL)
	if err != nil {
		t.Fatalf("connect restored database: %v", err)
	}
	defer target.Close()
	var restoredAmount string
	if err := target.QueryRow(ctx, `
		SELECT amount::text
		FROM transactions
		WHERE id = $1 AND account_id = $2 AND description = 'Backup sentinel'
	`, transactionID, accountID).Scan(&restoredAmount); err != nil {
		t.Fatalf("read restored financial data: %v", err)
	}
	if restoredAmount != "1234.560000000000000000" {
		t.Fatalf("restored amount = %q", restoredAmount)
	}
}

var execLookPath = exec.LookPath

func databaseURLWithName(t *testing.T, databaseURL, name string) string {
	t.Helper()
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parse database URL: %v", err)
	}
	parsed.Path = "/" + name
	return parsed.String()
}
