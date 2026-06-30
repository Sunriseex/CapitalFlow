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

func TestHTTPAdapterDoesNotComposeApplicationServices(t *testing.T) {
	walkGoFiles(t, "../http/handlers", func(path, content string) {
		if strings.HasSuffix(path, "_test.go") {
			return
		}
		if strings.Contains(content, "services.New") {
			t.Fatalf("%s composes an application service", path)
		}
	})
}

func TestHTTPAdapterDoesNotCallPersistence(t *testing.T) {
	forbidden := []string{
		".Store", ".Accounts()", ".Transactions()", ".Categories()",
		".FinancialGoals()", ".CategoryLimits()", ".InterestRules()",
		".Users()", ".RefreshTokens()", ".Idempotency()", ".Ping(",
	}
	walkGoFiles(t, "../http/handlers", func(path, content string) {
		if strings.HasSuffix(path, "_test.go") {
			return
		}
		for _, pattern := range forbidden {
			if strings.Contains(content, pattern) {
				t.Fatalf("%s calls persistence through %q", path, pattern)
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

func TestInterestCalculationUsesDedicatedEngine(t *testing.T) {
	data, err := os.ReadFile("../services/interest_rule_service.go")
	if err != nil {
		t.Fatalf("read interest module: %v", err)
	}
	content := string(data)
	for _, method := range []string{"Accrue", "Recalculate", "Forecast"} {
		if strings.Contains(content, "func (s *InterestRuleService) "+method) {
			t.Fatalf("interest rule management owns %s calculation", method)
		}
		if !strings.Contains(content, "func (e *InterestEngine) "+method) {
			t.Fatalf("interest engine does not own %s calculation", method)
		}
	}
	for _, root := range []string{"../services", "../application", "../jobs"} {
		walkGoFiles(t, root, func(path, content string) {
			if strings.Contains(content, "NewInterestRuleService(nil).") {
				t.Fatalf("%s uses configuration-dependent interest rule management", path)
			}
		})
	}
}

func TestTransactionQueriesDoNotFallBackToUnboundedReads(t *testing.T) {
	data, err := os.ReadFile("../services/transaction_query.go")
	if err != nil {
		t.Fatalf("read transaction query module: %v", err)
	}
	content := string(data)
	for _, forbidden := range []string{"applyTransactionListFilter", "ListByAccountForUser(ctx", ".(filteredTransactionLister)"} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("transaction query module contains fallback %q", forbidden)
		}
	}
	if !strings.Contains(content, "repository.TransactionQueryRepository") {
		t.Fatal("transaction query module does not use its bounded persistence seam")
	}
}

func TestAuthenticationPolicyOwnsSharedSecurityLifecycle(t *testing.T) {
	authData, err := os.ReadFile("../services/auth_service.go")
	if err != nil {
		t.Fatalf("read auth module: %v", err)
	}
	passkeyData, err := os.ReadFile("../services/passkey_service.go")
	if err != nil {
		t.Fatalf("read passkey module: %v", err)
	}
	policyData, err := os.ReadFile("../services/authentication_policy.go")
	if err != nil {
		t.Fatalf("read authentication policy: %v", err)
	}
	for path, content := range map[string]string{
		"auth service":    string(authData),
		"passkey service": string(passkeyData),
	} {
		for _, forbidden := range []string{"RecordLoginFailure(", "ClearLoginFailures(", "IssuePair(", "func (s *AuthService) auditEvent", "func (s *PasskeyService) auditEvent"} {
			if strings.Contains(content, forbidden) {
				t.Fatalf("%s owns shared authentication policy %q", path, forbidden)
			}
		}
	}
	passkeyContent := string(passkeyData)
	if strings.Contains(passkeyContent, "*AuthService") || strings.Contains(passkeyContent, "repository.AuthAuditRepository") {
		t.Fatal("passkey mechanism depends on auth workflow or audit persistence")
	}
	policyContent := string(policyData)
	for _, required := range []string{"ConfirmPassword(", "IssueSessionForUser(", "UserLocked(", "Audit("} {
		if !strings.Contains(policyContent, required) {
			t.Fatalf("authentication policy does not own %q", required)
		}
	}
}

func TestLegacyJSONIsOnlyReachableFromCommandModule(t *testing.T) {
	const legacyImport = "internal/" + "legacyjson"
	for _, root := range []string{"..", "../../cmd"} {
		walkGoFiles(t, root, func(path, content string) {
			if strings.Contains(path, filepath.Clean("../legacyjson")) {
				return
			}
			if !strings.Contains(content, legacyImport) {
				return
			}
			if filepath.Clean(path) != filepath.Clean("../application/commands.go") {
				t.Fatalf("%s imports the legacy JSON adapter", path)
			}
		})
	}
}

func TestCLIAdapterDoesNotComposeApplicationServices(t *testing.T) {
	walkGoFiles(t, "../../cmd/capitalflow", func(path, content string) {
		if strings.HasSuffix(path, "_test.go") {
			return
		}
		if strings.Contains(content, "services.New") {
			t.Fatalf("%s composes an application service", path)
		}
		for _, pattern := range []string{".Accounts()", ".Transactions()", ".InterestRules()", ".InterestAccruals()"} {
			if strings.Contains(content, pattern) {
				t.Fatalf("%s calls persistence through %q", path, pattern)
			}
		}
	})
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
