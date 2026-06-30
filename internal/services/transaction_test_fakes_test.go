package services

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/models"
)

type recordingTransactionRepo struct{}

func (*recordingTransactionRepo) Create(context.Context, *models.Transaction) error { return nil }
func (*recordingTransactionRepo) CreateForUser(context.Context, string, *models.Transaction) error {
	return errNotImplemented
}

func (*recordingTransactionRepo) CreateMany(context.Context, []models.Transaction) error { return nil }

func (*recordingTransactionRepo) CreateTransfer(context.Context, *models.Transfer, []models.Transaction) error {
	return nil
}

func (*recordingTransactionRepo) ListTransfersByUser(context.Context, string) ([]models.Transfer, error) {
	return nil, nil
}

func (*recordingTransactionRepo) GetByID(context.Context, string) (*models.Transaction, error) {
	return nil, errNotImplemented
}

func (*recordingTransactionRepo) GetByIDForUser(context.Context, string, string) (*models.Transaction, error) {
	return nil, errNotImplemented
}

func (*recordingTransactionRepo) List(context.Context) ([]models.Transaction, error) { return nil, nil }

func (*recordingTransactionRepo) ListByUser(context.Context, string) ([]models.Transaction, error) {
	return nil, nil
}

func (*recordingTransactionRepo) ListByAccount(context.Context, string) ([]models.Transaction, error) {
	return nil, nil
}

func (*recordingTransactionRepo) ListByAccountForUser(context.Context, string, string) ([]models.Transaction, error) {
	return nil, nil
}

func (*recordingTransactionRepo) GetBalanceByAccountForUser(context.Context, string, string) (decimal.Decimal, int64, error) {
	return decimal.Zero, 0, nil
}
