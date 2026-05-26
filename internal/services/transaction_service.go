package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	domaintransaction "github.com/sunriseex/capitalflow/internal/domain/transaction"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

type TransactionService struct {
	repo repository.TransactionRepository
}

var maxTransactionAmount = domaintransaction.MaxAmount

func NewTransactionService(repos ...repository.TransactionRepository) *TransactionService {
	var repo repository.TransactionRepository
	if len(repos) > 0 {
		repo = repos[0]
	}
	return &TransactionService{repo: repo}
}

type CreateTransactionRequest struct {
	AccountID        string
	RelatedAccountID *string
	Type             models.TransactionType
	Amount           decimal.Decimal
	Currency         string
	CategoryID       *string
	Description      string
	OccurredAt       time.Time
	AccountOpenedAt  time.Time
	AllowFutureDate  bool
}

func (s *TransactionService) Create(ctx context.Context, req *CreateTransactionRequest) (*models.Transaction, error) {
	transaction, err := buildTransaction(ctx, req)
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
	transaction, err := buildTransaction(ctx, req)
	if err != nil {
		return nil, err
	}

	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("transaction repository is required")
	}
	if err := s.repo.CreateForUser(ctx, strings.TrimSpace(userID), transaction); err != nil {
		return nil, fmt.Errorf("save transaction: %w", err)
	}

	return transaction, nil
}

func (s *TransactionService) CreateMany(ctx context.Context, reqs ...*CreateTransactionRequest) ([]models.Transaction, error) {
	transactions := make([]models.Transaction, 0, len(reqs))
	for _, req := range reqs {
		transaction, err := buildTransaction(ctx, req)
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
		transaction, err := buildTransaction(ctx, req)
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
	for i := range transactions[:2] {
		transactions[i].TransferID = &transfer.ID
	}
	if len(transactions) == 3 {
		transfer.FeeTransactionID = &transactions[2].ID
	}
	if err := s.repo.CreateTransfer(ctx, transfer, transactions); err != nil {
		return nil, fmt.Errorf("save transfer transactions: %w", err)
	}

	return transactions, nil
}

func buildTransaction(ctx context.Context, req *CreateTransactionRequest) (*models.Transaction, error) {
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
	}); err != nil {
		return nil, validationError(err.Error())
	}

	transaction := &models.Transaction{
		ID:               uuid.NewString(),
		AccountID:        strings.TrimSpace(req.AccountID),
		RelatedAccountID: req.RelatedAccountID,
		Type:             req.Type,
		Amount:           req.Amount,
		CategoryID:       req.CategoryID,
		Description:      strings.TrimSpace(req.Description),
		OccurredAt:       occurredAt,
		CreatedAt:        time.Now(),
	}

	return transaction, nil
}
