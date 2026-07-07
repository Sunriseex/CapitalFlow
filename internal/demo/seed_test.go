package demo

import (
	"testing"
	"time"
)

func TestValidateEnvironment(t *testing.T) {
	if err := ValidateEnvironment("development"); err != nil {
		t.Fatalf("development rejected: %v", err)
	}
	for _, environment := range []string{"", "production", "staging"} {
		if err := ValidateEnvironment(environment); err == nil {
			t.Fatalf("environment %q accepted", environment)
		}
	}
}

func TestBuildDatasetIsDeterministicAndCoversSixYears(t *testing.T) {
	now := time.Date(2026, 6, 27, 0, 0, 0, 0, time.UTC)
	categories := map[string]string{
		"salary": "salary", "food": "food", "transport": "transport",
		"subscriptions": "subscriptions", "housing": "housing", "health": "health",
		"entertainment": "entertainment", "deposit_interest": "interest", "other": "other",
		"travel": "travel", "clothing": "clothing", "gifts": "gifts", "education": "education",
	}
	first := buildDataset(now, categories, "rule")
	second := buildDataset(now, categories, "rule")
	if len(first.transactions) < 1200 || len(first.transactions) > 1450 {
		t.Fatalf("transactions = %d, want about 1300", len(first.transactions))
	}
	if len(first.transactions) != len(second.transactions) {
		t.Fatalf("transaction count changed: %d != %d", len(first.transactions), len(second.transactions))
	}
	oldest := now
	for i, transaction := range first.transactions {
		if transaction.occurredAt.Before(now.AddDate(-6, 0, 0)) {
			t.Fatalf("transaction %s is older than six years", transaction.id)
		}
		if transaction.occurredAt.After(now) {
			t.Fatalf("transaction %s is in the future", transaction.id)
		}
		if transaction.createdAt.Before(transaction.occurredAt) {
			t.Fatalf("transaction %s created before occurrence", transaction.id)
		}
		if transaction.id != second.transactions[i].id || !transaction.occurredAt.Equal(second.transactions[i].occurredAt) {
			t.Fatalf("transaction %d is not deterministic", i)
		}
		if transaction.occurredAt.Before(oldest) {
			oldest = transaction.occurredAt
		}
	}
	if !oldest.Equal(now.AddDate(-6, 0, 0)) {
		t.Fatalf("oldest transaction = %s, want %s", oldest, now.AddDate(-6, 0, 0))
	}
}

func TestCurrentMonthLimitRatios(t *testing.T) {
	now := time.Date(2026, 6, 27, 0, 0, 0, 0, time.UTC)
	categories := map[string]string{
		"salary": "salary", "food": "food", "transport": "transport",
		"subscriptions": "subscriptions", "housing": "housing", "health": "health",
		"entertainment": "entertainment", "deposit_interest": "interest", "other": "other",
		"travel": "travel", "clothing": "clothing", "gifts": "gifts", "education": "education",
	}
	data := buildDataset(now, categories, "rule")
	totals := map[string]string{}
	for _, transaction := range data.transactions {
		if transaction.transactionType == "expense" && transaction.categoryID != nil && transaction.occurredAt.Year() == now.Year() && transaction.occurredAt.Month() == now.Month() {
			totals[*transaction.categoryID] = transaction.amount
		}
	}
	for category, want := range map[string]string{"food": "45000", "transport": "8300", "subscriptions": "11000"} {
		if totals[category] != want {
			t.Fatalf("%s current amount = %s, want %s", category, totals[category], want)
		}
	}
}
