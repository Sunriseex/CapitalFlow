package models

import "time"

type IdempotencyRecord struct {
	ID           string
	Key          string
	UserID       string
	Method       string
	Path         string
	Endpoint     string
	RequestHash  string
	Status       string
	StatusCode   *int
	ResponseBody []byte
	LockedUntil  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ExpiresAt    time.Time
}
