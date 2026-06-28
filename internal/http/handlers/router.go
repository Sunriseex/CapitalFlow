package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/sunriseex/capitalflow/internal/auth"
	appmiddleware "github.com/sunriseex/capitalflow/internal/http/middleware"
	"github.com/sunriseex/capitalflow/internal/repository"
	"github.com/sunriseex/capitalflow/internal/services"
)

type Store interface {
	Accounts() repository.AccountRepository
	Transactions() repository.TransactionRepository
	Categories() repository.CategoryRepository
	FinancialGoals() repository.FinancialGoalRepository
	CategoryLimits() repository.CategoryLimitRepository
	InterestRules() repository.InterestRuleRepository
	InterestAccruals() repository.InterestAccrualRepository
	Users() repository.UserRepository
	RefreshTokens() repository.RefreshTokenRepository
	AuthAuditEvents() repository.AuthAuditRepository
	Passkeys() repository.PasskeyRepository
	Idempotency() repository.IdempotencyRepository
	Ping(ctx context.Context) error
}

type Handler struct {
	store                 Store
	appVersion            string
	tokens                *auth.TokenService
	cookieSecure          bool
	cookieSameSite        http.SameSite
	webAuthnRPDisplayName string
	webAuthnRPID          string
	webAuthnOrigins       []string
	auth                  *services.AuthService
	passkeys              *services.PasskeyService
	accounts              *services.AccountService
	transactions          *services.TransactionService
	transfers             *services.TransferService
	interestRules         *services.InterestRuleService
}

type RouterConfig struct {
	AppEnv                          string
	AppVersion                      string
	APIAuthToken                    string
	TokenService                    *auth.TokenService
	PublicOrigin                    string
	PublicOriginHost                string
	WebAuthnRPDisplayName           string
	WebAuthnRPID                    string
	WebAuthnOrigins                 []string
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
}

func NewRouter(store Store, cfg *RouterConfig) http.Handler {
	if cfg == nil {
		cfg = &RouterConfig{}
	}

	var accountRepo repository.AccountRepository
	var transactionRepo repository.TransactionRepository
	var categoryRepo repository.CategoryRepository
	var interestRuleRepo repository.InterestRuleRepository
	var interestAccrualRepo repository.InterestAccrualRepository
	if store != nil {
		accountRepo = store.Accounts()
		transactionRepo = store.Transactions()
		categoryRepo = store.Categories()
		interestRuleRepo = store.InterestRules()
		interestAccrualRepo = store.InterestAccruals()
	}

	transactionService := services.NewTransactionService(transactionRepo).
		WithAccountRepository(accountRepo).
		WithCategoryRepository(categoryRepo)
	authService := newRouterAuthService(store, cfg.TokenService)
	passkeyService := newRouterPasskeyService(store, authService, cfg)
	cookieSecure := cfg.CookieSecure
	if cfg.AppEnv == "" && cfg.CookieSameSite == "" {
		cookieSecure = true
	}
	h := &Handler{
		store:                 store,
		appVersion:            firstNonEmpty(cfg.AppVersion, "v0.5.8"),
		tokens:                cfg.TokenService,
		cookieSecure:          cookieSecure,
		cookieSameSite:        cookieSameSiteMode(cfg.CookieSameSite),
		webAuthnRPDisplayName: firstNonEmpty(cfg.WebAuthnRPDisplayName, "CapitalFlow"),
		webAuthnRPID:          firstNonEmpty(cfg.WebAuthnRPID, "localhost"),
		webAuthnOrigins:       defaultWebAuthnOrigins(cfg.WebAuthnOrigins),
		auth:                  authService,
		passkeys:              passkeyService,
		accounts:              services.NewAccountService(accountRepo),
		transactions:          transactionService,
		transfers:             services.NewTransferService(transactionService),
		interestRules: services.NewInterestRuleService(
			transactionService,
			services.WithInterestRuleRepository(interestRuleRepo),
			services.WithInterestAccrualRepository(interestAccrualRepo),
		),
	}
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
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
		if cfg.TokenService != nil {
			r.Use(appmiddleware.JWTAuth(cfg.TokenService, h.store.RefreshTokens()))
		} else {
			r.Use(appmiddleware.BearerTokenAuth(cfg.APIAuthToken))
		}

		r.With(appmiddleware.MutationOnly(mutationRateLimit), appmiddleware.Idempotency(h.idempotency())).Group(func(r chi.Router) {
			r.Post("/auth/password", h.changePassword)
			r.Delete("/auth/sessions/{id}", h.revokeSession)
			r.Post("/auth/passkeys/registration/options", h.passkeyRegistrationOptions)
			r.Post("/auth/passkeys/registration/verify", h.passkeyRegistrationVerify)
			r.Patch("/auth/passkeys/{id}", h.renamePasskey)
			r.Delete("/auth/passkeys/{id}", h.deletePasskey)
			r.Patch("/settings/profile", h.updateProfile)
			r.Patch("/financial-goals/{id}", h.updateFinancialGoal)
			r.Patch("/category-limits/{id}", h.updateCategoryLimit)
			r.Post("/accounts", h.createAccount)
			r.Patch("/accounts/{id}", h.updateAccount)
			r.Post("/accounts/{id}/archive", h.archiveAccount)
			r.With(appmiddleware.RequireIdempotencyKey).Post("/transactions", h.createTransaction)
			r.With(appmiddleware.RequireIdempotencyKey).Post("/transfers", h.createTransfer)
			r.Post("/accounts/{id}/interest-rules", h.createInterestRule)
			r.Patch("/interest-rules/{id}", h.updateInterestRule)
			r.With(appmiddleware.RequireIdempotencyKey).Post("/accounts/{id}/accrue-interest", h.accrueInterest)
			r.With(appmiddleware.RequireIdempotencyKey).Post("/accounts/{id}/recalculate-interest", h.recalculateInterest)
		})

		r.Get("/categories", h.listCategories)
		r.With(appmiddleware.RequireIdempotencyKey).Post("/categories", h.createCategory)
		r.Get("/financial-goals", h.listFinancialGoals)
		r.With(appmiddleware.RequireIdempotencyKey).Post("/financial-goals", h.createFinancialGoal)
		r.Get("/category-limits", h.listCategoryLimits)
		r.With(appmiddleware.RequireIdempotencyKey).Post("/category-limits", h.createCategoryLimit)
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

func newRouterAuthService(store Store, tokens *auth.TokenService) *services.AuthService {
	if store == nil {
		return nil
	}
	return services.NewAuthService(
		store.Users(),
		store.RefreshTokens(),
		tokens,
		store.AuthAuditEvents(),
	).WithAccountRepository(store.Accounts())
}

func newRouterPasskeyService(store Store, authService *services.AuthService, cfg *RouterConfig) *services.PasskeyService {
	if store == nil || authService == nil || cfg.TokenService == nil {
		return nil
	}
	service, err := services.NewPasskeyService(
		store.Users(),
		store.Passkeys(),
		authService,
		store.AuthAuditEvents(),
		services.WebAuthnConfig{
			RPDisplayName: firstNonEmpty(cfg.WebAuthnRPDisplayName, "CapitalFlow"),
			RPID:          firstNonEmpty(cfg.WebAuthnRPID, "localhost"),
			Origins:       defaultWebAuthnOrigins(cfg.WebAuthnOrigins),
		},
	)
	if err != nil {
		panic("passkey service is not configured: " + err.Error())
	}
	return service
}

func firstNonEmpty(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func defaultWebAuthnOrigins(origins []string) []string {
	if len(origins) > 0 {
		return origins
	}
	return []string{"http://localhost:5173"}
}

func (h *Handler) idempotency() repository.IdempotencyRepository {
	if h.store == nil {
		return nil
	}
	return h.store.Idempotency()
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
