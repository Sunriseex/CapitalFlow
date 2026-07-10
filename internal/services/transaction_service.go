package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	domaintransaction "github.com/sunriseex/capitalflow/internal/domain/transaction"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

type TransactionService struct {
	repo       repository.TransactionRepository
	accounts   repository.AccountRepository
	categories repository.CategoryRepository
}

var maxTransactionAmount = domaintransaction.MaxAmount

func NewTransactionService(repos ...repository.TransactionRepository) *TransactionService {
	var repo repository.TransactionRepository
	if len(repos) > 0 {
		repo = repos[0]
	}
	return &TransactionService{repo: repo}
}

func (s *TransactionService) WithAccountRepository(repo repository.AccountRepository) *TransactionService {
	s.accounts = repo
	return s
}

func (s *TransactionService) WithCategoryRepository(repo repository.CategoryRepository) *TransactionService {
	s.categories = repo
	return s
}

type CreateTransactionRequest struct {
	AccountID        string
	RelatedAccountID *string
	Type             models.TransactionType
	Amount           decimal.Decimal
	Currency         string
	CategoryID       *string
	SourceType       models.TransactionSource
	SourceRefID      *string
	SourceMetadata   json.RawMessage
	Description      string
	OccurredAt       time.Time
	AccountOpenedAt  time.Time
	AllowFutureDate  bool
}

func (s *TransactionService) Create(ctx context.Context, req *CreateTransactionRequest) (*models.Transaction, error) {
	req, err := s.resolveCreateRequest(ctx, "", req)
	if err != nil {
		return nil, err
	}
	transaction, err := buildTransaction(ctx, req, false)
	if err != nil {
		return nil, err
	}

	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("transaction repository is required")
	}
	if err := s.repo.Create(ctx, transaction); err != nil {
		return nil, fmt.Errorf("save transaction: %w", err)
	}

	return transaction, nil
}

func (s *TransactionService) CreateForUser(ctx context.Context, userID string, req *CreateTransactionRequest) (*models.Transaction, error) {
	userID = strings.TrimSpace(userID)
	req, err := s.resolveCreateRequest(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	transaction, err := buildTransaction(ctx, req, false)
	if err != nil {
		return nil, err
	}

	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("transaction repository is required")
	}
	if err := s.repo.CreateForUser(ctx, userID, transaction); err != nil {
		return nil, fmt.Errorf("save transaction: %w", err)
	}

	return transaction, nil
}

func (s *TransactionService) CreateMany(ctx context.Context, reqs ...*CreateTransactionRequest) ([]models.Transaction, error) {
	transactions := make([]models.Transaction, 0, len(reqs))
	for _, req := range reqs {
		req, err := s.resolveCreateRequest(ctx, "", req)
		if err != nil {
			return nil, err
		}
		transaction, err := buildTransaction(ctx, req, false)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, *transaction)
	}

	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("transaction repository is required")
	}
	if err := s.repo.CreateMany(ctx, transactions); err != nil {
		return nil, fmt.Errorf("save transactions: %w", err)
	}

	return transactions, nil
}

func (s *TransactionService) CreateTransfer(ctx context.Context, transfer *models.Transfer, reqs ...*CreateTransactionRequest) ([]models.Transaction, error) {
	transactions := make([]models.Transaction, 0, len(reqs))
	for _, req := range reqs {
		transaction, err := buildTransaction(ctx, req, true)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, *transaction)
	}

	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("transaction repository is required")
	}
	if transfer == nil {
		return nil, validationError("transfer audit record is required")
	}
	if len(transactions) != 2 && len(transactions) != 3 {
		return nil, validationError("transfer requires two transactions and optional fee transaction")
	}
	transfer.FromTransactionID = transactions[0].ID
	transfer.ToTransactionID = transactions[1].ID
	for i := range transactions {
		transactions[i].SourceType = models.TransactionSourceTransfer
		transactions[i].SourceRefID = &transfer.ID
		if i < 2 {
			transactions[i].TransferID = &transfer.ID
		}
	}
	if len(transactions) == 3 {
		transfer.FeeTransactionID = &transactions[2].ID
	}
	if err := s.repo.CreateTransfer(ctx, transfer, transactions); err != nil {
		return nil, fmt.Errorf("save transfer transactions: %w", err)
	}

	return transactions, nil
}

func (s *TransactionService) CancelForUser(ctx context.Context, userID, transactionID string) (*models.Transaction, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("transaction repository is required")
	}
	transaction, err := s.repo.CancelForUser(ctx, strings.TrimSpace(transactionID), strings.TrimSpace(userID))
	if err != nil {
		return nil, fmt.Errorf("cancel transaction: %w", err)
	}
	return transaction, nil
}

func (s *TransactionService) SoftDeleteForUser(ctx context.Context, userID, transactionID string) (*models.Transaction, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("transaction repository is required")
	}
	transaction, err := s.repo.SoftDeleteForUser(ctx, strings.TrimSpace(transactionID), strings.TrimSpace(userID))
	if err != nil {
		return nil, fmt.Errorf("soft-delete transaction: %w", err)
	}
	return transaction, nil
}

func (s *TransactionService) ReverseForUser(ctx context.Context, userID, transactionID string) (updated, created *models.Transaction, err error) {
	if s == nil || s.repo == nil {
		return nil, nil, fmt.Errorf("transaction repository is required")
	}
	original, err := s.repo.GetByIDForUser(ctx, strings.TrimSpace(transactionID), strings.TrimSpace(userID))
	if err != nil {
		return nil, nil, fmt.Errorf("get transaction for reversal: %w", err)
	}
	if original.Status != "" && original.Status != models.TransactionStatusConfirmed {
		return nil, nil, validationError("only confirmed transactions can be reversed")
	}
	if original.SourceType != "" && original.SourceType != models.TransactionSourceManual {
		return nil, nil, validationError("only manual transactions can be reversed")
	}
	if original.TransferID != nil {
		return nil, nil, validationError("transfer transactions cannot be reversed through this endpoint")
	}

	now := time.Now()
	reversalTransaction := &models.Transaction{
		ID:             uuid.NewString(),
		AccountID:      original.AccountID,
		SourceType:     models.TransactionSourceReconciliationAdjustment,
		SourceRefID:    &original.ID,
		SourceMetadata: json.RawMessage(`{}`),
		Type:           models.TransactionTypeAdjustment,
		Status:         models.TransactionStatusConfirmed,
		Amount:         transactionDeltaForReversal(original).Neg(),
		Description:    "Reversal of transaction " + original.ID,
		OccurredAt:     now,
		CreatedAt:      now,
	}
	updated, created, err = s.repo.ReverseForUser(ctx, strings.TrimSpace(transactionID), strings.TrimSpace(userID), reversalTransaction)
	if err != nil {
		return nil, nil, fmt.Errorf("reverse transaction: %w", err)
	}
	return updated, created, nil
}

func (s *TransactionService) resolveCreateRequest(ctx context.Context, userID string, req *CreateTransactionRequest) (*CreateTransactionRequest, error) {
	if req == nil {
		return nil, fmt.Errorf("create transaction request is required")
	}
	resolved := *req
	resolved.AccountID = strings.TrimSpace(req.AccountID)

	if s != nil && s.accounts != nil && resolved.AccountID != "" {
		var (
			account *models.Account
			err     error
		)
		if userID == "" {
			account, err = s.accounts.GetByID(ctx, resolved.AccountID)
		} else {
			account, err = s.accounts.GetByIDForUser(ctx, resolved.AccountID, userID)
		}
		if err != nil {
			return nil, fmt.Errorf("get transaction account: %w", err)
		}
		resolved.Currency = account.Currency
		resolved.AccountOpenedAt = account.OpenedAt
	}

	if resolved.RelatedAccountID != nil {
		relatedAccountID := strings.TrimSpace(*resolved.RelatedAccountID)
		if relatedAccountID == "" {
			resolved.RelatedAccountID = nil
		} else {
			if s != nil && s.accounts != nil {
				if userID == "" {
					if _, err := s.accounts.GetByID(ctx, relatedAccountID); err != nil {
						return nil, fmt.Errorf("get related transaction account: %w", err)
					}
				} else if _, err := s.accounts.GetByIDForUser(ctx, relatedAccountID, userID); err != nil {
					return nil, fmt.Errorf("get related transaction account: %w", err)
				}
			}
			resolved.RelatedAccountID = &relatedAccountID
		}
	}

	if resolved.CategoryID != nil {
		categoryID := strings.TrimSpace(*resolved.CategoryID)
		if categoryID == "" {
			resolved.CategoryID = nil
		} else {
			if s != nil && s.categories != nil {
				if _, err := s.categories.GetByID(ctx, categoryID); err != nil {
					return nil, fmt.Errorf("get transaction category: %w", err)
				}
			}
			resolved.CategoryID = &categoryID
		}
	}

	return &resolved, nil
}

func buildTransaction(ctx context.Context, req *CreateTransactionRequest, allowTransfer bool) (*models.Transaction, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("create transaction: %w", ctx.Err())
	default:
	}
	if req == nil {
		return nil, fmt.Errorf("create transaction request is required")
	}

	occurredAt := req.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}
	if err := domaintransaction.ValidateCreate(&domaintransaction.CreateValidation{
		AccountID:     req.AccountID,
		Type:          req.Type,
		Amount:        req.Amount,
		Currency:      req.Currency,
		OccurredAt:    occurredAt,
		AccountOpened: req.AccountOpenedAt,
		AllowFuture:   req.AllowFutureDate,
		AllowTransfer: allowTransfer,
	}); err != nil {
		return nil, validationError(err.Error())
	}

	sourceType := req.SourceType
	if sourceType == "" {
		sourceType = models.TransactionSourceManual
	}
	if !sourceType.IsValid() {
		return nil, validationError("invalid transaction source type")
	}
	var sourceRefID *string
	if req.SourceRefID != nil {
		normalized := strings.TrimSpace(*req.SourceRefID)
		if _, err := uuid.Parse(normalized); err != nil {
			return nil, validationError("transaction source reference must be a UUID")
		}
		sourceRefID = &normalized
	}
	if len(req.SourceMetadata) > 0 {
		metadata := bytes.TrimSpace(req.SourceMetadata)
		if !json.Valid(metadata) || len(metadata) == 0 || metadata[0] != '{' {
			return nil, validationError("transaction source metadata must be a JSON object")
		}
	}
	sourceMetadata := slices.Clone(req.SourceMetadata)
	if len(sourceMetadata) == 0 {
		sourceMetadata = json.RawMessage(`{}`)
	}
	transaction := &models.Transaction{
		ID:               uuid.NewString(),
		AccountID:        strings.TrimSpace(req.AccountID),
		RelatedAccountID: req.RelatedAccountID,
		SourceType:       sourceType,
		SourceRefID:      sourceRefID,
		SourceMetadata:   sourceMetadata,
		Type:             req.Type,
		Status:           models.TransactionStatusConfirmed,
		Amount:           req.Amount,
		CategoryID:       req.CategoryID,
		Description:      strings.TrimSpace(req.Description),
		OccurredAt:       occurredAt,
		CreatedAt:        time.Now(),
	}

	return transaction, nil
}

func transactionDeltaForReversal(tx *models.Transaction) decimal.Decimal {
	switch tx.Type {
	case models.TransactionTypeInitialBalance,
		models.TransactionTypeIncome,
		models.TransactionTypeTransferIn,
		models.TransactionTypeInterestIncome,
		models.TransactionTypeAdjustment:
		return tx.Amount
	case models.TransactionTypeExpense,
		models.TransactionTypeTransferOut:
		return tx.Amount.Neg()
	default:
		return decimal.Zero
	}
}
