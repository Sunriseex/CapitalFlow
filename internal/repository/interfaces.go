package repository

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/models"
)

type TransactionListFilter struct {
	AccountID  string
	CategoryID string
	Type       models.TransactionType
	FromDate   time.Time
	ToDate     time.Time
	Search     string
	Limit      int
	Page       int
}

type AccountRepository interface {
	Create(ctx context.Context, account *models.Account) error
	GetByID(ctx context.Context, id string) (*models.Account, error)
	GetByIDForUser(ctx context.Context, id, userID string) (*models.Account, error)
	GetByLegacyID(ctx context.Context, legacyID string) (*models.Account, error)
	List(ctx context.Context) ([]models.Account, error)
	ListByUser(ctx context.Context, userID string) ([]models.Account, error)
	Update(ctx context.Context, account *models.Account) error
	UpdateForUser(ctx context.Context, account *models.Account, userID string) error
	UpdateForUserEnforcingCurrencyInvariant(ctx context.Context, account *models.Account, userID string) error
	Archive(ctx context.Context, id string) error
	ArchiveForUser(ctx context.Context, id, userID string) error
	ClaimUnowned(ctx context.Context, userID string) error
}

type DepositMigrationRepository interface {
	CreateMigratedDeposit(ctx context.Context, account *models.Account, rule *models.InterestRule, transaction *models.Transaction) error
}

type TransactionRepository interface {
	Create(ctx context.Context, transaction *models.Transaction) error
	CreateForUser(ctx context.Context, userID string, transaction *models.Transaction) error
	CreateMany(ctx context.Context, transactions []models.Transaction) error
	CreateTransfer(ctx context.Context, transfer *models.Transfer, transactions []models.Transaction) error
	ListTransfersByUser(ctx context.Context, userID string) ([]models.Transfer, error)
	GetByID(ctx context.Context, id string) (*models.Transaction, error)
	GetByIDForUser(ctx context.Context, id, userID string) (*models.Transaction, error)
	List(ctx context.Context) ([]models.Transaction, error)
	ListByUser(ctx context.Context, userID string) ([]models.Transaction, error)
	ListByAccount(ctx context.Context, accountID string) ([]models.Transaction, error)
	ListByAccountForUser(ctx context.Context, accountID, userID string) ([]models.Transaction, error)
	GetBalanceByAccountForUser(ctx context.Context, accountID, userID string) (balance decimal.Decimal, transactionCount int64, err error)
}

type CategoryRepository interface {
	Create(ctx context.Context, category *models.Category) error
	GetByID(ctx context.Context, id string) (*models.Category, error)
	GetBySlug(ctx context.Context, slug string) (*models.Category, error)
	List(ctx context.Context) ([]models.Category, error)
}

type InterestRuleRepository interface {
	Create(ctx context.Context, rule *models.InterestRule) error
	GetByID(ctx context.Context, id string) (*models.InterestRule, error)
	ListByAccount(ctx context.Context, accountID string) ([]models.InterestRule, error)
	Update(ctx context.Context, rule *models.InterestRule) error
}

type InterestRuleJobTarget struct {
	Rule            models.InterestRule
	OwnerUserID     string
	AccountCurrency string
}

type InterestRuleJobRepository interface {
	ListActiveForAccrual(ctx context.Context, frequency models.AccrualFrequency, accrualDate time.Time) ([]InterestRuleJobTarget, error)
}

type InterestAccrualRepository interface {
	Create(ctx context.Context, accrual *models.InterestAccrual) error
	CreateWithTransaction(ctx context.Context, transaction *models.Transaction, accrual *models.InterestAccrual) error
	ReplaceRangeWithTransactions(ctx context.Context, accountID, ruleID string, fromDate, toDate time.Time, transactions []models.Transaction, accruals []models.InterestAccrual) (int64, error)
	GetByAccountDateRule(ctx context.Context, accountID, accrualDate, ruleID string) (*models.InterestAccrual, error)
	ListByAccount(ctx context.Context, accountID string) ([]models.InterestAccrual, error)
}

type InterestCalculationRepository interface {
	GetInterestRuleByID(ctx context.Context, id string) (*models.InterestRule, error)
	ListInterestRulesByAccount(ctx context.Context, accountID string) ([]models.InterestRule, error)
	ListTransactionsByAccountForUser(ctx context.Context, accountID, userID string) ([]models.Transaction, error)
	ListInterestAccrualsByAccount(ctx context.Context, accountID string) ([]models.InterestAccrual, error)
	CreateInterestAccrualWithTransaction(ctx context.Context, transaction *models.Transaction, accrual *models.InterestAccrual) error
	ReplaceInterestAccrualRangeWithTransactions(ctx context.Context, accountID, ruleID string, fromDate, toDate time.Time, transactions []models.Transaction, accruals []models.InterestAccrual) (int64, error)
}

type InterestAccrualTransactionalRepository interface {
	WithAccountInterestLock(ctx context.Context, accountID, userID string, fn func(context.Context, InterestCalculationRepository) error) error
}

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	Count(ctx context.Context) (int64, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByID(ctx context.Context, id string) (*models.User, error)
	RecordLoginFailure(ctx context.Context, id string, threshold int, delays []time.Duration, updatedAt time.Time) (int, *time.Time, error)
	ClearLoginFailures(ctx context.Context, id string, updatedAt time.Time) error
	UpdatePassword(ctx context.Context, id, passwordHash string, updatedAt time.Time) error
	ChangePasswordAndRevokeSessions(ctx context.Context, id, passwordHash string, updatedAt time.Time, revokedReason string) error
	UpdatePrimaryCurrency(ctx context.Context, id, primaryCurrency string, updatedAt time.Time) error
}

type AuthSetupRepository interface {
	Setup(ctx context.Context, user *models.User, refreshToken *models.RefreshToken, auditEvent *models.AuthAuditEvent) error
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *models.RefreshToken) error
	GetByID(ctx context.Context, id string) (*models.RefreshToken, error)
	GetByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error)
	ListByUser(ctx context.Context, userID string) ([]models.RefreshToken, error)
	Revoke(ctx context.Context, id string, revokedAt time.Time, reason string) error
	RevokeByUserSession(ctx context.Context, userID, id string, revokedAt time.Time, reason string) error
	RevokeByUser(ctx context.Context, userID string, revokedAt time.Time, reason string) error
}

type AuthAuditRepository interface {
	Create(ctx context.Context, event *models.AuthAuditEvent) error
}

// PasskeyRepository persists WebAuthn passkey credentials and one-use challenges.
type PasskeyRepository interface {
	CreateCredential(ctx context.Context, credential *models.PasskeyCredential) error
	ListCredentialsByUser(ctx context.Context, userID string, includeRevoked bool) ([]models.PasskeyCredential, error)
	GetCredentialByIDForUser(ctx context.Context, id, userID string) (*models.PasskeyCredential, error)
	GetCredentialByCredentialID(ctx context.Context, credentialID []byte) (*models.PasskeyCredential, error)
	CountActiveCredentialsByUser(ctx context.Context, userID string) (int64, error)
	UpdateCredentialAfterLogin(ctx context.Context, credentialID []byte, signCount uint32, cloneWarning, backupState bool, lastUsedAt time.Time) error
	RenameCredential(ctx context.Context, id, userID, name string, updatedAt time.Time) error
	RevokeCredential(ctx context.Context, id, userID string, revokedAt time.Time) error
	CreateChallenge(ctx context.Context, challenge *models.WebAuthnChallenge) error
	ConsumeChallenge(ctx context.Context, ceremony, challenge string, userID *string, usedAt time.Time) (*models.WebAuthnChallenge, error)
}

type IdempotencyRepository interface {
	Get(ctx context.Context, key, userID, method, path string) (*models.IdempotencyRecord, error)
	CreatePending(ctx context.Context, record *models.IdempotencyRecord) (bool, error)
	Complete(ctx context.Context, key, userID, method, path string, statusCode int, responseBody []byte) error
}
