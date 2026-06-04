package repository

import (
	"context"
	"time"

	"github.com/sunriseex/capitalflow/internal/models"
)

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
	DeleteExpiredChallenges(ctx context.Context, before time.Time) error
}
