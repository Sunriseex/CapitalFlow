package commands

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/sunriseex/capitalflow/internal/config"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/storage"
	"github.com/sunriseex/capitalflow/pkg/errors"
	"github.com/sunriseex/capitalflow/pkg/security"
)

type ExportSnapshot struct {
	ExportedAt time.Time            `json:"exported_at"`
	Source     ExportSnapshotSource `json:"source"`
	Deposits   []models.Deposit     `json:"deposits"`
	Payments   []models.Payment     `json:"payments"`
}

type ExportSnapshotSource struct {
	AppVersion string `json:"app_version"`
}

func DepositExport(outputPath string) error {
	if outputPath == "" {
		outputPath = "capitalflow-export-" + time.Now().UTC().Format("20060102T150405Z") + ".json"
	}

	deposits, err := storage.LoadDeposits(config.AppConfig.DepositsDataPath)
	if err != nil {
		return errors.NewStorageError("экспорт вкладов", err)
	}

	payments, err := storage.LoadPaymentsOrEmpty(config.AppConfig.DataPath)
	if err != nil {
		return errors.NewStorageError("экспорт платежей", err)
	}

	snapshot := ExportSnapshot{
		ExportedAt: time.Now().UTC(),
		Source: ExportSnapshotSource{
			AppVersion: config.AppConfig.AppVersion,
		},
		Deposits: deposits.Deposits,
		Payments: payments.Payments,
	}

	if err := security.AtomicWriteJSON(snapshot, outputPath); err != nil {
		return errors.NewStorageError("запись export файла", err)
	}

	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		absPath = outputPath
	}
	fmt.Printf("✅ Export saved: %s\n", absPath)
	return nil
}

func DepositBackup() error {
	targets := []struct {
		name string
		path string
	}{
		{name: "deposits", path: config.AppConfig.DepositsDataPath},
		{name: "payments", path: config.AppConfig.DataPath},
	}

	created := 0
	for _, target := range targets {
		backupPath, err := security.BackupFile(target.path)
		if err != nil {
			return errors.NewStorageError("backup "+target.name, err)
		}
		if backupPath == "" {
			fmt.Printf("SKIP %s: source not found\n", target.name)
			continue
		}
		created++
		fmt.Printf("OK   %s: %s\n", target.name, backupPath)
	}

	if created == 0 {
		fmt.Println("No files were backed up")
		return nil
	}

	fmt.Printf("✅ Backup completed: %d file(s)\n", created)
	return nil
}
