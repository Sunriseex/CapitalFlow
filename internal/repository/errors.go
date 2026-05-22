package repository

import "errors"

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")
var ErrAccountCurrencyInvariant = errors.New("account currency cannot be changed after transactions exist")
var ErrInsufficientFunds = errors.New("insufficient funds")
