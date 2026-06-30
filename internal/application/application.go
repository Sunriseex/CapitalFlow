package application

import (
	"context"
	"fmt"

	"github.com/sunriseex/capitalflow/internal/auth"
	"github.com/sunriseex/capitalflow/internal/repository"
	"github.com/sunriseex/capitalflow/internal/services"
)

// Store is the persistence boundary used to compose the application.
type Store interface {
	Accounts() repository.AccountRepository
	Transactions() repository.TransactionRepository
	TransactionQueries() repository.TransactionQueryRepository
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

type Config struct {
	TokenService          *auth.TokenService
	WebAuthnRPDisplayName string
	WebAuthnRPID          string
	WebAuthnOrigins       []string
}

// Application owns service composition. Transport adapters receive this
// ready-to-use module and never construct services themselves.
type Application struct {
	Tokens             *auth.TokenService
	Auth               *services.AuthService
	Authentication     *services.AuthenticationPolicy
	Passkeys           *services.PasskeyService
	Accounts           *services.AccountService
	Transactions       *services.TransactionService
	TransactionQueries *services.TransactionQuery
	Transfers          *services.TransferService
	InterestRules      *services.InterestRuleService
	InterestEngine     *services.InterestEngine
	InterestLifecycle  *services.InterestLifecycle
	Dashboard          *services.DashboardReporting
	Profile            *services.ProfileService
	Currency           *services.CurrencyService
	Categories         *services.CategoryService
	FinancialGoals     *services.FinancialGoalService
	CategoryLimits     *services.CategoryLimitService
	Commands           *CommandModule
	Idempotency        repository.IdempotencyRepository
	readiness          interface{ Ping(context.Context) error }
}

func New(store Store, cfg Config) (*Application, error) {
	var accountRepo repository.AccountRepository
	var transactionRepo repository.TransactionRepository
	var transactionQueryRepo repository.TransactionQueryRepository
	var categoryRepo repository.CategoryRepository
	var interestRuleRepo repository.InterestRuleRepository
	var interestAccrualRepo repository.InterestAccrualRepository
	var interestLifecycleRepo repository.InterestAccrualTransactionalRepository
	var dashboardRepo repository.DashboardRepository
	var userRepo repository.UserRepository

	if store != nil {
		accountRepo = store.Accounts()
		transactionRepo = store.Transactions()
		transactionQueryRepo = store.TransactionQueries()
		categoryRepo = store.Categories()
		interestRuleRepo = store.InterestRules()
		interestAccrualRepo = store.InterestAccruals()
		interestLifecycleRepo, _ = interestAccrualRepo.(repository.InterestAccrualTransactionalRepository)
		userRepo = store.Users()
		if dashboardStore, ok := store.(interface {
			Dashboard() repository.DashboardRepository
		}); ok {
			dashboardRepo = dashboardStore.Dashboard()
		}
	}

	transactions := services.NewTransactionService(transactionRepo).
		WithAccountRepository(accountRepo).
		WithCategoryRepository(categoryRepo)
	interestEngine := services.NewInterestEngine()
	app := &Application{
		Tokens:             cfg.TokenService,
		Accounts:           services.NewAccountService(accountRepo).WithTransactionRepository(transactionRepo),
		Transactions:       transactions,
		TransactionQueries: services.NewTransactionQuery(transactionQueryRepo),
		Transfers:          services.NewTransferService(transactions).WithAccountRepository(accountRepo),
		InterestRules:      services.NewInterestRuleService(interestRuleRepo, accountRepo),
		InterestEngine:     interestEngine,
		InterestLifecycle:  services.NewInterestLifecycle(interestLifecycleRepo, interestEngine).WithAccountRepository(accountRepo),
		Dashboard:          services.NewDashboardReporting(dashboardRepo),
		Profile:            services.NewProfileService(userRepo),
		Currency:           services.NewCurrencyService(nil),
		Categories:         services.NewCategoryService(categoryRepo),
		FinancialGoals:     services.NewFinancialGoalService(storeFinancialGoals(store), accountRepo),
		CategoryLimits:     services.NewCategoryLimitService(storeCategoryLimits(store), categoryRepo),
		readiness:          store,
	}
	app.Commands = newCommandModule(store, app)
	if store == nil {
		return app, nil
	}
	app.Idempotency = store.Idempotency()

	app.Auth = services.NewAuthService(store.Users(), store.RefreshTokens(), cfg.TokenService, store.AuthAuditEvents()).
		WithAccountRepository(accountRepo)
	app.Authentication = app.Auth.AuthenticationPolicy()
	if cfg.TokenService == nil {
		return app, nil
	}

	passkeys, err := services.NewPasskeyService(
		store.Users(),
		store.Passkeys(),
		app.Authentication,
		services.WebAuthnConfig{
			RPDisplayName: firstNonEmpty(cfg.WebAuthnRPDisplayName, "CapitalFlow"),
			RPID:          firstNonEmpty(cfg.WebAuthnRPID, "localhost"),
			Origins:       defaultWebAuthnOrigins(cfg.WebAuthnOrigins),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("configure passkey service: %w", err)
	}
	app.Passkeys = passkeys
	return app, nil
}

func (a *Application) Ready(ctx context.Context) error {
	if a == nil || a.readiness == nil {
		return fmt.Errorf("readiness check is not configured")
	}
	if err := a.readiness.Ping(ctx); err != nil {
		return fmt.Errorf("application readiness: %w", err)
	}
	return nil
}

func storeFinancialGoals(store Store) repository.FinancialGoalRepository {
	if store == nil {
		return nil
	}
	return store.FinancialGoals()
}

func storeCategoryLimits(store Store) repository.CategoryLimitRepository {
	if store == nil {
		return nil
	}
	return store.CategoryLimits()
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
