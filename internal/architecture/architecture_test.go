package architecture

import (
	"fmt"
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

func TestInterestAdaptersDoNotOwnTransactionalOrchestration(t *testing.T) {
	forbidden := []string{
		"WithAccountInterestLock(",
		"InterestCalculationRepository",
		"PrincipalTransactionsForRuleAt(",
		"ReplaceInterestAccrualRangeWithTransactions(",
	}
	for _, path := range []string{"../http/handlers/interest_rules.go", "../jobs/interest.go"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, pattern := range forbidden {
			if strings.Contains(string(data), pattern) {
				t.Fatalf("%s owns interest lifecycle detail %q", path, pattern)
			}
		}
	}
}

func TestLegacyJSONIsOnlyReachableFromImportCLI(t *testing.T) {
	const legacyImport = "internal/" + "legacyjson"
	for _, root := range []string{"..", "../../cmd"} {
		walkGoFiles(t, root, func(path, content string) {
			if strings.Contains(path, filepath.Clean("../legacyjson")) {
				return
			}
			if !strings.Contains(content, legacyImport) {
				return
			}
			if filepath.Clean(path) != filepath.Clean("../../cmd/capitalflow/main.go") {
				t.Fatalf("%s imports the legacy JSON adapter", path)
			}
		})
	}
}

func TestTransactionHardDeleteIsLimitedToGeneratedInterestReplacement(t *testing.T) {
	allowed := map[string]bool{
		filepath.Clean("../postgres/interest_accruals.go"): true,
		filepath.Clean("../demo/seed.go"):                  true,
	}

	walkGoFiles(t, "..", func(path, content string) {
		if strings.HasSuffix(path, "_test.go") {
			return
		}
		if !strings.Contains(content, "DELETE FROM transactions") {
			return
		}
		if !allowed[filepath.Clean(path)] {
			t.Fatalf("%s contains transaction hard delete outside generated interest replacement", path)
		}
	})
}

func walkGoFiles(t *testing.T, root string, fn func(path, content string)) {
	t.Helper()

	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", path, err)
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		paths = append(paths, filepath.Clean(path))
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		fn(filepath.Clean(path), string(data))
	}
}
