package repository

import "errors"

var (
	ErrNotFound                 = errors.New("not found")
	ErrConflict                 = errors.New("conflict")
	ErrAccountCurrencyInvariant = errors.New("account currency cannot be changed after transactions exist")
	ErrInsufficientFunds        = errors.New("insufficient funds")
)
