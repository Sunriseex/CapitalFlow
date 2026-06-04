package models

import "time"

// PasskeyCredential stores a WebAuthn credential owned by a user.
type PasskeyCredential struct {
	ID              string
	UserID          string
	CredentialID    []byte
	PublicKey       []byte
	AttestationType string
	Transports      []string
	SignCount       uint32
	CloneWarning    bool
	BackupEligible  bool
	BackupState     bool
	Name            string
	AAGUID          *string
	LastUsedAt      *time.Time
	RevokedAt       *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// IsActive reports whether the credential can be used for login.
func (c *PasskeyCredential) IsActive() bool {
	return c.RevokedAt == nil
}

// WebAuthnChallenge stores the server-side state for a WebAuthn ceremony.
type WebAuthnChallenge struct {
	ID          string
	UserID      *string
	Ceremony    string
	Challenge   string
	SessionData []byte
	ExpiresAt   time.Time
	UsedAt      *time.Time
	CreatedAt   time.Time
}
