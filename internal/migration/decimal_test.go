package migration

import "github.com/shopspring/decimal"

func dec(value string) decimal.Decimal {
	return decimal.RequireFromString(value)
}
