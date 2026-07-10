package models

import "testing"

func TestTransactionStatusValidationAndBalanceEffect(t *testing.T) {
	tests := []struct {
		status         TransactionStatus
		valid          bool
		affectsBalance bool
	}{
		{status: "", affectsBalance: true},
		{status: TransactionStatusPending, valid: true},
		{status: TransactionStatusConfirmed, valid: true, affectsBalance: true},
		{status: TransactionStatusCancelled, valid: true},
		{status: TransactionStatusReversed, valid: true, affectsBalance: true},
		{status: TransactionStatusSoftDeleted, valid: true},
		{status: TransactionStatus("bad")},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.valid {
				t.Fatalf("IsValid = %t, want %t", got, tt.valid)
			}
			if got := tt.status.AffectsBalance(); got != tt.affectsBalance {
				t.Fatalf("AffectsBalance = %t, want %t", got, tt.affectsBalance)
			}
		})
	}
}
