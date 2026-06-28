package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/sunriseex/capitalflow/internal/demo"
	"github.com/sunriseex/capitalflow/internal/postgres"
	"github.com/sunriseex/capitalflow/pkg/security"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	reset := flag.Bool("reset", false, "remove the local demo user and its data")
	flag.Parse()
	_ = godotenv.Load("./configs/.env")
	if err := demo.ValidateEnvironment(os.Getenv("APP_ENV")); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = "postgres://capitalflow:capitalflow@localhost:5432/capitalflow?sslmode=disable"
	}
	pool, err := postgres.OpenPool(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if *reset {
		if err := demo.Reset(ctx, pool); err != nil {
			return err
		}
		fmt.Println("demo data removed")
		return nil
	}
	password := os.Getenv("DEMO_PASSWORD")
	if len(password) < 12 {
		return fmt.Errorf("DEMO_PASSWORD must contain at least 12 characters")
	}
	hash, err := security.HashPassword(password, security.DefaultPasswordParams())
	if err != nil {
		return err
	}
	result, err := demo.Seed(ctx, pool, hash, time.Now())
	if err != nil {
		return err
	}
	fmt.Printf("demo ready: %s, accounts=%d transactions=%d transfers=%d goals=%d limits=%d\n", demo.Email, result.Accounts, result.Transactions, result.Transfers, result.Goals, result.Limits)
	return nil
}
