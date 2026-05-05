package commands

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/sunriseex/finance-manager/internal/config"
	"github.com/sunriseex/finance-manager/internal/models"
	"github.com/sunriseex/finance-manager/pkg/security"
)

func DepositDoctor() error {
	startedAt := time.Now()
	slog.Info("doctor started")
	checks := []struct {
		name string
		err  error
	}{
		{name: "deposits JSON", err: validateDepositsFile(config.AppConfig.DepositsDataPath)},
		{name: "payments JSON", err: validatePaymentsFile(config.AppConfig.DataPath)},
	}

	failed := 0
	for _, check := range checks {
		if check.err != nil {
			failed++
			slog.Warn("doctor check failed", "check", check.name, "error", check.err)
			fmt.Printf("FAIL %s: %v\n", check.name, check.err)
			continue
		}
		slog.Info("doctor check passed", "check", check.name)
		fmt.Printf("OK   %s\n", check.name)
	}

	if failed > 0 {
		slog.Warn("doctor completed with failures", "failed", failed, "duration", time.Since(startedAt))
		return fmt.Errorf("doctor found %d failed checks", failed)
	}

	slog.Info("doctor completed", "duration", time.Since(startedAt))
	fmt.Println("Doctor completed successfully")
	return nil
}

func validateDepositsFile(path string) error {
	var data models.DepositsData
	if err := security.SafeReadJSON(path, &data); err != nil {
		return err
	}
	for _, deposit := range data.Deposits {
		if strings.TrimSpace(deposit.ID) == "" {
			return fmt.Errorf("deposit has empty id")
		}
		if deposit.Amount < 0 {
			return fmt.Errorf("deposit %s has negative amount", deposit.ID)
		}
		if strings.TrimSpace(deposit.Name) == "" {
			return fmt.Errorf("deposit %s has empty name", deposit.ID)
		}
	}
	return nil
}

func validatePaymentsFile(path string) error {
	var data models.PaymentData
	if err := security.SafeReadJSON(path, &data); err != nil {
		return err
	}
	for _, payment := range data.Payments {
		if strings.TrimSpace(payment.ID) == "" {
			return fmt.Errorf("payment has empty id")
		}
		if payment.Amount <= 0 {
			return fmt.Errorf("payment %s has non-positive amount", payment.ID)
		}
		if strings.TrimSpace(payment.Name) == "" {
			return fmt.Errorf("payment %s has empty name", payment.ID)
		}
	}
	return nil
}

func validateWritableDirectory(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory")
	}
	return nil
}
