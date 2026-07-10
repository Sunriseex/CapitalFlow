package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/sunriseex/capitalflow/internal/application"
	appmiddleware "github.com/sunriseex/capitalflow/internal/http/middleware"
)

type Handler struct {
	app                  *application.Application
	appVersion           string
	cookieSecure         bool
	cookieSameSite       http.SameSite
	dbPoolMetrics        func() DBPoolMetrics
	operationsMetricsDir string
}

type RouterConfig struct {
	AppEnv                          string
	AppVersion                      string
	APIAuthToken                    string
	PublicOrigin                    string
	PublicOriginHost                string
	CookieSecure                    bool
	CookieSameSite                  string
	AllowDirectIPLogin              bool
	CORSAllowedOrigins              []string
	RateLimitRequests               int
	RateLimitWindow                 time.Duration
	AuthRateLimitRequests           int
	AuthRateLimitWindow             time.Duration
	PasskeyOptionsRateLimitRequests int
	PasskeyOptionsRateLimitWindow   time.Duration
	MutationRateLimitRequests       int
	MutationRateLimitWindow         time.Duration
	TrustedProxies                  []string
	DBPoolMetrics                   func() DBPoolMetrics
	OperationsMetricsDir            string
}

const maxRequestBodyBytes = 1 << 20

func NewRouter(app *application.Application, cfg *RouterConfig) http.Handler {
	if cfg == nil {
		cfg = &RouterConfig{}
	}
	if app == nil {
		panic("application is required")
	}
	cookieSecure := cfg.CookieSecure
	if cfg.AppEnv == "" && cfg.CookieSameSite == "" {
		cookieSecure = true
	}
	h := &Handler{
		app:                  app,
		appVersion:           firstNonEmpty(cfg.AppVersion, "v0.5.8"),
		cookieSecure:         cookieSecure,
		cookieSameSite:       cookieSameSiteMode(cfg.CookieSameSite),
		dbPoolMetrics:        cfg.DBPoolMetrics,
		operationsMetricsDir: cfg.OperationsMetricsDir,
	}
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(appmiddleware.RequestMetrics)
	r.Use(appmiddleware.LimitRequestBody(maxRequestBodyBytes))
	r.Use(appmiddleware.RequestLogger)
	r.Use(chimiddleware.Recoverer)
	r.Use(appmiddleware.SecurityHeaders(appmiddleware.SecurityHeadersConfig{
		PublicOrigin: cfg.PublicOrigin,
		CookieSecure: cfg.CookieSecure,
	}))
	r.Use(appmiddleware.CORS(&appmiddleware.CORSConfig{
		AllowedOrigins: cfg.CORSAllowedOrigins,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders:   []string{"Authorization", "Content-Type", appmiddleware.IdempotencyKeyHeader},
		AllowCredentials: true,
	}))
	r.Use(appmiddleware.AuthHostPolicy(appmiddleware.HostPolicyConfig{
		AppEnv:             cfg.AppEnv,
		PublicOrigin:       cfg.PublicOrigin,
		PublicOriginHost:   cfg.PublicOriginHost,
		AllowDirectIPLogin: cfg.AllowDirectIPLogin,
	}))

	authRateLimit := appmiddleware.RateLimitByIPWithTrustedProxies(
		firstPositive(cfg.AuthRateLimitRequests, cfg.RateLimitRequests),
		firstPositiveDuration(cfg.AuthRateLimitWindow, cfg.RateLimitWindow),
		cfg.TrustedProxies,
	)
	passkeyOptionsRateLimit := appmiddleware.RateLimitByIPWithTrustedProxies(
		firstPositive(cfg.PasskeyOptionsRateLimitRequests, cfg.AuthRateLimitRequests),
		firstPositiveDuration(cfg.PasskeyOptionsRateLimitWindow, cfg.AuthRateLimitWindow),
		cfg.TrustedProxies,
	)

	mutationRateLimit := appmiddleware.RateLimitByIPWithTrustedProxies(
		firstPositive(cfg.MutationRateLimitRequests, cfg.RateLimitRequests),
		firstPositiveDuration(cfg.MutationRateLimitWindow, cfg.RateLimitWindow),
		cfg.TrustedProxies,
	)

	r.Get("/health", h.health)
	r.Get("/ready", h.ready)
	r.Get("/metrics", h.metrics)
	r.Get("/auth/status", h.authStatus)
	r.With(authRateLimit).Post("/auth/setup", h.authSetup)
	r.With(authRateLimit).Post("/auth/login", h.authLogin)
	r.With(authRateLimit).Post("/auth/refresh", h.authRefresh)
	r.With(authRateLimit).Post("/auth/logout", h.authLogout)
	r.With(passkeyOptionsRateLimit).Post("/auth/passkeys/login/options", h.passkeyLoginOptions)
	r.With(authRateLimit).Post("/auth/passkeys/login/verify", h.passkeyLoginVerify)

	r.Route("/api/v1", func(r chi.Router) {
		if app.Tokens != nil {
			r.Use(appmiddleware.JWTAuth(app.Tokens, app.Auth))
		} else {
			r.Use(appmiddleware.BearerTokenAuth(cfg.APIAuthToken))
		}

		r.With(appmiddleware.MutationOnly(mutationRateLimit), appmiddleware.Idempotency(app.Idempotency)).Group(func(r chi.Router) {
			r.Post("/auth/password", h.changePassword)
			r.Delete("/auth/sessions/{id}", h.revokeSession)
			r.Post("/auth/passkeys/registration/options", h.passkeyRegistrationOptions)
			r.Post("/auth/passkeys/registration/verify", h.passkeyRegistrationVerify)
			r.Patch("/auth/passkeys/{id}", h.renamePasskey)
			r.Delete("/auth/passkeys/{id}", h.deletePasskey)
			r.Patch("/settings/profile", h.updateProfile)
			r.Patch("/financial-goals/{id}", h.updateFinancialGoal)
			r.Patch("/category-limits/{id}", h.updateCategoryLimit)
			r.With(appmiddleware.RequireIdempotencyKey).Post("/categories", h.createCategory)
			r.With(appmiddleware.RequireIdempotencyKey).Post("/financial-goals", h.createFinancialGoal)
			r.With(appmiddleware.RequireIdempotencyKey).Post("/category-limits", h.createCategoryLimit)
			r.Post("/accounts", h.createAccount)
			r.Patch("/accounts/{id}", h.updateAccount)
			r.Post("/accounts/{id}/archive", h.archiveAccount)
			r.With(appmiddleware.RequireIdempotencyKey).Post("/transactions", h.createTransaction)
			r.With(appmiddleware.RequireIdempotencyKey).Post("/transactions/{id}/cancel", h.cancelTransaction)
			r.With(appmiddleware.RequireIdempotencyKey).Post("/transactions/{id}/reverse", h.reverseTransaction)
			r.With(appmiddleware.RequireIdempotencyKey).Post("/transactions/{id}/soft-delete", h.softDeleteTransaction)
			r.With(appmiddleware.RequireIdempotencyKey).Post("/transfers", h.createTransfer)
			r.Post("/accounts/{id}/interest-rules", h.createInterestRule)
			r.Patch("/interest-rules/{id}", h.updateInterestRule)
			r.With(appmiddleware.RequireIdempotencyKey).Post("/accounts/{id}/accrue-interest", h.accrueInterest)
			r.With(appmiddleware.RequireIdempotencyKey).Post("/accounts/{id}/recalculate-interest", h.recalculateInterest)
		})

		r.Get("/categories", h.listCategories)
		r.Get("/financial-goals", h.listFinancialGoals)
		r.Get("/category-limits", h.listCategoryLimits)
		r.Get("/auth/sessions", h.listSessions)
		r.Get("/auth/passkeys", h.listPasskeys)
		r.Get("/currency-rates", h.getCurrencyRates)
		r.Get("/settings/profile", h.getProfile)

		r.Get("/accounts", h.listAccounts)
		r.Get("/accounts/{id}", h.getAccount)
		r.Get("/accounts/{id}/balance", h.getAccountBalance)

		r.Get("/transactions", h.listTransactions)
		r.Get("/transactions/{id}", h.getTransaction)
		r.Get("/transfers", h.listTransfers)

		r.Get("/interest-rules", h.listUserInterestRules)
		r.Get("/accounts/{id}/interest-rules", h.listInterestRules)

		r.Get("/dashboard/summary", h.getDashboardSummary)
		r.Get("/dashboard/net-worth", h.getDashboardNetWorth)
		r.Get("/dashboard/cashflow", h.getDashboardCashflow)
		r.Get("/dashboard/interest-income", h.getDashboardInterestIncome)
	})

	return r
}

func firstNonEmpty(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func firstPositive(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func firstPositiveDuration(value, fallback time.Duration) time.Duration {
	if value > 0 {
		return value
	}
	return fallback
}
