package handlers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOperationMetricsReadSchedulerState(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{
		"capitalflow-backup-scheduler.heartbeat",
		"capitalflow-backup-scheduler.last-success",
	} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte("ok\n"), 0o600); err != nil {
			t.Fatalf("write operation metric: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(directory, "capitalflow-backup-scheduler.status"), []byte("status=success\n"), 0o600); err != nil {
		t.Fatalf("write operation status: %v", err)
	}

	backup := schedulerMetrics(directory, "backup")
	if backup["status"] != "success" {
		t.Fatalf("backup status = %v, want success", backup["status"])
	}
	if backup["heartbeat_age_seconds"].(int64) < 0 || backup["last_success_age_seconds"].(int64) < 0 {
		t.Fatalf("backup ages = %+v, want non-negative", backup)
	}

	interest := schedulerMetrics(directory, "interest")
	if interest["status"] != "unknown" || interest["last_success_age_seconds"] != int64(-1) {
		t.Fatalf("missing interest metrics = %+v", interest)
	}
}
