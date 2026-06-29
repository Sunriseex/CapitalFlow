// Package legacyjson imports read-only snapshots from the predecessor deposit tracker.
package legacyjson

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultDepositSnapshotPath = "~/.config/waybar/deposits.json"

	DepositTypeSavings = "savings"
	DepositTypeTerm    = "term"

	CapitalizationDaily   = "daily"
	CapitalizationMonthly = "monthly"
	CapitalizationEnd     = "end"
)

type Deposit struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Bank           string    `json:"bank"`
	Type           string    `json:"type"`
	Amount         int64     `json:"amount"`
	InitialAmount  int64     `json:"initial_amount"`
	InterestRate   float64   `json:"interest_rate"`
	PromoRate      *float64  `json:"promo_rate,omitempty"`
	PromoEndDate   string    `json:"promo_end_date,omitempty"`
	StartDate      string    `json:"start_date"`
	EndDate        string    `json:"end_date,omitempty"`
	TermMonths     int       `json:"term_months,omitempty"`
	Capitalization string    `json:"capitalization"`
	AutoRenewal    bool      `json:"auto_renewal"`
	TopUpEndDate   string    `json:"top_up_end_date,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Snapshot struct {
	Deposits []Deposit `json:"deposits"`
}

func Load(path string) (*Snapshot, error) {
	path, err := expandPath(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read legacy deposit snapshot: %w", err)
	}
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("decode legacy deposit snapshot: %w", err)
	}
	if snapshot.Deposits == nil {
		snapshot.Deposits = []Deposit{}
	}
	return &snapshot, nil
}

func expandPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("legacy deposit snapshot path is required")
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve legacy deposit snapshot path: %w", err)
	}
	return abs, nil
}
