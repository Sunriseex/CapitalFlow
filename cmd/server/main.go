package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sunriseex/capitalflow/internal/auth"
	"github.com/sunriseex/capitalflow/internal/config"
	"github.com/sunriseex/capitalflow/internal/http/handlers"
	"github.com/sunriseex/capitalflow/internal/postgres"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := config.Init(); err != nil {
		return fmt.Errorf("config init: %w", err)
	}

	addr := flag.String("addr", ":8080", "HTTP listen address")
	databaseURL := flag.String("database-url", config.AppConfig.DatabaseURL, "PostgreSQL connection URL")
	flag.Parse()

	if err := config.ValidateAuthSecret("JWT_SECRET", config.AppConfig.JWTSecret); err != nil {
		return err
	}

	pool, err := postgres.OpenPool(ctx, *databaseURL)
	if err != nil {
		return fmt.Errorf("open postgres pool: %w", err)
	}
	defer pool.Close()

	store := postgres.NewStore(pool)
	if err := store.Ping(ctx); err != nil {
		return fmt.Errorf("postgres ping: %w", err)
	}

	tokenService, err := auth.NewTokenService(
		config.AppConfig.JWTSecret,
		"capitalflow",
		config.AppConfig.AccessTokenTTL,
		config.AppConfig.RefreshTokenTTL,
	)
	if err != nil {
		return fmt.Errorf("init token service: %w", err)
	}
	slog.Info("webauthn configured",
		"rp_id", config.AppConfig.WebAuthnRPID,
		"origins", config.AppConfig.WebAuthnOrigins)

	server := &http.Server{
		Addr: *addr,
		Handler: handlers.NewRouter(store, &handlers.RouterConfig{
			AppEnv:                          config.AppConfig.AppEnv,
			AppVersion:                      config.AppConfig.AppVersion,
			APIAuthToken:                    config.AppConfig.APIAuthToken,
			TokenService:                    tokenService,
			PublicOrigin:                    config.AppConfig.PublicOrigin,
			PublicOriginHost:                config.AppConfig.PublicOriginHost,
			WebAuthnRPDisplayName:           config.AppConfig.WebAuthnRPDisplayName,
			WebAuthnRPID:                    config.AppConfig.WebAuthnRPID,
			WebAuthnOrigins:                 config.AppConfig.WebAuthnOrigins,
			CookieSecure:                    config.AppConfig.CookieSecure,
			CookieSameSite:                  config.AppConfig.CookieSameSite,
			AllowDirectIPLogin:              config.AppConfig.AllowDirectIPLogin,
			CORSAllowedOrigins:              config.AppConfig.CORSAllowedOrigins,
			RateLimitRequests:               config.AppConfig.RateLimitRequests,
			RateLimitWindow:                 config.AppConfig.RateLimitWindow,
			AuthRateLimitRequests:           config.AppConfig.AuthRateLimitRequests,
			AuthRateLimitWindow:             config.AppConfig.AuthRateLimitWindow,
			PasskeyOptionsRateLimitRequests: config.AppConfig.PasskeyOptionsRateLimitRequests,
			PasskeyOptionsRateLimitWindow:   config.AppConfig.PasskeyOptionsRateLimitWindow,
			MutationRateLimitRequests:       config.AppConfig.MutationRateLimitRequests,
			MutationRateLimitWindow:         config.AppConfig.MutationRateLimitWindow,
			TrustedProxies:                  config.AppConfig.TrustedProxies,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErr := make(chan error, 1)

	go func() {
		slog.Info("server listening", "addr", *addr)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("listen and serve: %w", err)
			return
		}

		serverErr <- nil
	}()

	select {
	case err := <-serverErr:
		return err

	case <-ctx.Done():
		slog.Info("shutdown signal received")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return fmt.Errorf("shutdown server: %w", err)
		}

		if err := <-serverErr; err != nil {
			return err
		}

		return nil
	}
}
