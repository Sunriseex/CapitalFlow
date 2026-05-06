package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/sunriseex/finance-manager/internal/config"
	"github.com/sunriseex/finance-manager/internal/http/handlers"
	"github.com/sunriseex/finance-manager/internal/postgres"
)

func main() {
	if err := config.Init(); err != nil {
		slog.Error("config init failed", "error", err)
		os.Exit(1)
	}

	addr := flag.String("addr", ":8080", "HTTP listen address")
	databaseURL := flag.String("database-url", config.AppConfig.DatabaseURL, "PostgreSQL connection URL")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := postgres.OpenPool(ctx, *databaseURL)
	if err != nil {
		slog.Error("open postgres failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	store := postgres.NewStore(pool)
	if err := store.Ping(ctx); err != nil {
		slog.Error("postgres ping failed", "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:              *addr,
		Handler:           handlers.NewRouter(store),
		ReadHeaderTimeout: 5 * time.Second,
	}

	slog.Info("server listening", "addr", *addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
