package legacyjson

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsSnapshotWithoutMutatingIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deposits.json")
	data := []byte(`{"deposits":[{"id":"legacy-1","name":"Reserve","amount":10000}]}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}

	snapshot, err := Load(path)
	if err != nil {
		t.Fatalf("load snapshot: %v", err)
	}
	if len(snapshot.Deposits) != 1 || snapshot.Deposits[0].ID != "legacy-1" {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read snapshot after load: %v", err)
	}
	if !bytes.Equal(after, data) {
		t.Fatal("load mutated the source snapshot")
	}
}

func TestLoadDoesNotCreateMissingSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	if _, err := Load(path); err == nil {
		t.Fatal("load missing snapshot: error = nil")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("missing snapshot was created: %v", err)
	}
}
