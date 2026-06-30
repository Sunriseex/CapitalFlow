package main

import (
	"context"
	"strings"
	"testing"

	"github.com/sunriseex/capitalflow/internal/config"
)

func TestRunTransactionsCreateRejectsTransferTypes(t *testing.T) {
	oldConfig := config.AppConfig
	config.AppConfig = &config.Config{
		DatabaseURL: "postgres://test:test@localhost:5432/test?sslmode=disable",
	}
	t.Cleanup(func() {
		config.AppConfig = oldConfig
	})

	tests := []struct {
		name            string
		transactionType string
	}{
		{
			name:            "transfer in",
			transactionType: "transfer_in",
		},
		{
			name:            "transfer out",
			transactionType: "transfer_out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runTransactionsCreate(context.Background(), []string{
				"--account", "account-1",
				"--type", tt.transactionType,
				"--amount", "100.00",
			})

			if err == nil {
				t.Fatal("expected error")
			}

			if !strings.Contains(err.Error(), "transfer transactions") {
				t.Fatalf("error = %q, want transfer rejection", err.Error())
			}
		})
	}
}

func TestRunJobsRunRejectsUnknownJobBeforeOpeningDatabase(t *testing.T) {
	oldConfig := config.AppConfig
	config.AppConfig = &config.Config{
		DatabaseURL: "postgres://invalid:invalid@127.0.0.1:1/invalid?sslmode=disable",
	}
	t.Cleanup(func() {
		config.AppConfig = oldConfig
	})

	err := runJobsRun(context.Background(), []string{"--name", "unknown_job"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unknown job name: unknown_job") {
		t.Fatalf("error = %q, want unknown job rejection", err.Error())
	}
}

func TestValidInterestJobName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: "daily_interest_accrual_job", want: true},
		{name: "monthly_interest_accrual_job", want: true},
		{name: "deposit_maturity_check_job", want: true},
		{name: "unknown_job", want: false},
		{name: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validInterestJobName(tt.name); got != tt.want {
				t.Fatalf("validInterestJobName(%q) = %t, want %t", tt.name, got, tt.want)
			}
		})
	}
}
