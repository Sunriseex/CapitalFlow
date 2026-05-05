package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sunriseex/finance-manager/internal/config"
	"github.com/sunriseex/finance-manager/internal/models"
	"github.com/sunriseex/finance-manager/pkg/security"
)

func TestDepositExportWritesSnapshot(t *testing.T) {
	tmp := t.TempDir()
	setTestConfig(tmp)

	deposits := models.DepositsData{
		Deposits: []models.Deposit{{
			ID:             "deposit-1",
			Name:           "Savings",
			Bank:           "Yandex",
			Type:           "savings",
			Amount:         100_000,
			InitialAmount:  100_000,
			InterestRate:   12,
			StartDate:      "2026-05-04",
			Capitalization: "daily",
		}},
	}
	if err := security.AtomicWriteJSON(deposits, config.AppConfig.DepositsDataPath); err != nil {
		t.Fatalf("write deposits: %v", err)
	}
	if err := security.AtomicWriteJSON(models.PaymentData{}, config.AppConfig.DataPath); err != nil {
		t.Fatalf("write payments: %v", err)
	}

	exportPath := filepath.Join(tmp, "export.json")
	if err := DepositExport(exportPath); err != nil {
		t.Fatalf("export: %v", err)
	}

	data, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	var snapshot ExportSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("decode export: %v", err)
	}
	if len(snapshot.Deposits) != 1 {
		t.Fatalf("deposits count = %d, want 1", len(snapshot.Deposits))
	}
	if snapshot.Source.AppVersion != "test" {
		t.Fatalf("app version = %q, want test", snapshot.Source.AppVersion)
	}
}

func TestDepositExportTreatsMissingPaymentsFileAsEmpty(t *testing.T) {
	tmp := t.TempDir()
	setTestConfig(tmp)

	if err := security.AtomicWriteJSON(models.DepositsData{}, config.AppConfig.DepositsDataPath); err != nil {
		t.Fatalf("write deposits: %v", err)
	}

	exportPath := filepath.Join(tmp, "export.json")
	if err := DepositExport(exportPath); err != nil {
		t.Fatalf("export: %v", err)
	}

	data, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	var snapshot ExportSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("decode export: %v", err)
	}
	if len(snapshot.Payments) != 0 {
		t.Fatalf("payments count = %d, want 0", len(snapshot.Payments))
	}
}

func TestExecuteNoArgsTreatsMissingPaymentsFileAsEmpty(t *testing.T) {
	tmp := t.TempDir()
	setTestConfig(tmp)

	oldArgs := os.Args
	os.Args = []string{"payments-manager"}
	t.Cleanup(func() {
		os.Args = oldArgs
	})

	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = writer

	err = Execute()

	writer.Close()
	os.Stdout = oldStdout

	var output bytes.Buffer
	if _, copyErr := output.ReadFrom(reader); copyErr != nil {
		t.Fatalf("read output: %v", copyErr)
	}
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if output.String() != "Нет активных платежей\n" {
		t.Fatalf("output = %q, want empty payment message", output.String())
	}
}

func TestDepositBackupCreatesFiles(t *testing.T) {
	tmp := t.TempDir()
	setTestConfig(tmp)

	if err := security.AtomicWriteJSON(models.DepositsData{}, config.AppConfig.DepositsDataPath); err != nil {
		t.Fatalf("write deposits: %v", err)
	}
	if err := security.AtomicWriteJSON(models.PaymentData{}, config.AppConfig.DataPath); err != nil {
		t.Fatalf("write payments: %v", err)
	}

	if err := DepositBackup(); err != nil {
		t.Fatalf("backup: %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(tmp, "backups"))
	if err != nil {
		t.Fatalf("read backups: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("backup count = %d, want 2", len(entries))
	}
}

func setTestConfig(tmp string) {
	config.AppConfig = &config.Config{
		AppVersion:       "test",
		DataPath:         filepath.Join(tmp, "payments.json"),
		DepositsDataPath: filepath.Join(tmp, "deposits.json"),
	}
}
