package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServicesAndRepositoriesDoNotImportHTTPDTO(t *testing.T) {
	for _, root := range []string{"../services", "../repository"} {
		walkGoFiles(t, root, func(path, content string) {
			if strings.Contains(content, "internal/http/dto") {
				t.Fatalf("%s imports HTTP DTOs", path)
			}
		})
	}
}

func TestHandlersDoNotBypassFinancialServices(t *testing.T) {
	forbidden := []string{
		".Transactions().Create(",
		".Transactions().CreateForUser(",
		".Transactions().CreateMany(",
		".Transactions().CreateTransfer(",
		".Transactions().Delete(",
		".Transactions().DeleteForUser(",
		".Accounts().UpdateForUserEnforcingCurrencyInvariant(",
		".Accounts().ArchiveForUser(",
	}

	walkGoFiles(t, "../http/handlers", func(path, content string) {
		for _, pattern := range forbidden {
			if strings.Contains(content, pattern) {
				t.Fatalf("%s bypasses service boundary with %q", path, pattern)
			}
		}
	})
}

func TestTransactionHardDeleteIsLimitedToGeneratedInterestReplacement(t *testing.T) {
	allowed := filepath.Clean("../postgres/interest_accruals.go")

	walkGoFiles(t, "..", func(path, content string) {
		if strings.HasSuffix(path, "_test.go") {
			return
		}
		if !strings.Contains(content, "DELETE FROM transactions") {
			return
		}
		if filepath.Clean(path) != allowed {
			t.Fatalf("%s contains transaction hard delete outside generated interest replacement", path)
		}
	})
}

func walkGoFiles(t *testing.T, root string, fn func(path, content string)) {
	t.Helper()

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		fn(filepath.Clean(path), string(data))
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
}
