package dto

import "time"

// PasskeyRegistrationOptionsRequest starts a passkey registration ceremony.
type PasskeyRegistrationOptionsRequest struct {
	Password string `json:"password"`
}

// PasskeyCredentialResponse is a user-visible passkey record.
type PasskeyCredentialResponse struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	BackupEligible bool       `json:"backup_eligible"`
	BackupState    bool       `json:"backup_state"`
	LastUsedAt     *time.Time `json:"last_used_at,omitzero"`
	CreatedAt      time.Time  `json:"created_at"`
}

// PasskeyCredentialsResponse lists user-visible passkeys.
type PasskeyCredentialsResponse struct {
	Passkeys []PasskeyCredentialResponse `json:"passkeys"`
}

// PasskeyRenameRequest renames a passkey.
type PasskeyRenameRequest struct {
	Name string `json:"name"`
}
