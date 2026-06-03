package dto

import "time"

type PasskeyRegistrationOptionsRequest struct {
	Password string `json:"password"`
}

type PasskeyCredentialResponse struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	BackupEligible bool       `json:"backup_eligible"`
	BackupState    bool       `json:"backup_state"`
	LastUsedAt     *time.Time `json:"last_used_at,omitzero"`
	CreatedAt      time.Time  `json:"created_at"`
}

type PasskeyCredentialsResponse struct {
	Passkeys []PasskeyCredentialResponse `json:"passkeys"`
}

type PasskeyRenameRequest struct {
	Name string `json:"name"`
}
