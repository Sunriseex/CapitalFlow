package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

// PasskeyRepository stores passkey credentials and WebAuthn challenges in PostgreSQL.
type PasskeyRepository struct {
	pool *pgxpool.Pool
}

// NewPasskeyRepository creates a PostgreSQL passkey repository.
func NewPasskeyRepository(pool *pgxpool.Pool) *PasskeyRepository {
	return &PasskeyRepository{pool: pool}
}

func (r *PasskeyRepository) CreateCredential(ctx context.Context, credential *models.PasskeyCredential) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO passkey_credentials (
			id, user_id, credential_id, public_key, attestation_type, transports,
			sign_count, clone_warning, backup_eligible, backup_state, name, aaguid,
			last_used_at, revoked_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, credential.ID, credential.UserID, credential.CredentialID, credential.PublicKey, credential.AttestationType,
		credential.Transports, credential.SignCount, credential.CloneWarning, credential.BackupEligible,
		credential.BackupState, credential.Name, credential.AAGUID, credential.LastUsedAt, credential.RevokedAt,
		credential.CreatedAt, credential.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create passkey credential: %w", mapConflict(err))
	}
	return nil
}

func (r *PasskeyRepository) ListCredentialsByUser(ctx context.Context, userID string, includeRevoked bool) ([]models.PasskeyCredential, error) {
	query := `
		SELECT id, user_id, credential_id, public_key, attestation_type, transports,
			sign_count, clone_warning, backup_eligible, backup_state, name, aaguid,
			last_used_at, revoked_at, created_at, updated_at
		FROM passkey_credentials
		WHERE user_id = $1`
	if !includeRevoked {
		query += ` AND revoked_at IS NULL`
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list passkey credentials: %w", err)
	}
	defer rows.Close()

	credentials := []models.PasskeyCredential{}
	for rows.Next() {
		credential, err := scanPasskeyCredential(rows)
		if err != nil {
			return nil, err
		}
		credentials = append(credentials, *credential)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list passkey credentials rows: %w", err)
	}
	return credentials, nil
}

func (r *PasskeyRepository) GetCredentialByIDForUser(ctx context.Context, id, userID string) (*models.PasskeyCredential, error) {
	credential, err := scanPasskeyCredential(r.pool.QueryRow(ctx, `
		SELECT id, user_id, credential_id, public_key, attestation_type, transports,
			sign_count, clone_warning, backup_eligible, backup_state, name, aaguid,
			last_used_at, revoked_at, created_at, updated_at
		FROM passkey_credentials
		WHERE id = $1 AND user_id = $2
	`, id, userID))
	if err != nil {
		return nil, fmt.Errorf("get passkey credential: %w", mapNotFound(err))
	}
	return credential, nil
}

func (r *PasskeyRepository) GetCredentialByCredentialID(ctx context.Context, credentialID []byte) (*models.PasskeyCredential, error) {
	credential, err := scanPasskeyCredential(r.pool.QueryRow(ctx, `
		SELECT id, user_id, credential_id, public_key, attestation_type, transports,
			sign_count, clone_warning, backup_eligible, backup_state, name, aaguid,
			last_used_at, revoked_at, created_at, updated_at
		FROM passkey_credentials
		WHERE credential_id = $1
	`, credentialID))
	if err != nil {
		return nil, fmt.Errorf("get passkey credential by credential id: %w", mapNotFound(err))
	}
	return credential, nil
}

func (r *PasskeyRepository) CountActiveCredentialsByUser(ctx context.Context, userID string) (int64, error) {
	var count int64
	if err := r.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM passkey_credentials
		WHERE user_id = $1 AND revoked_at IS NULL
	`, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count passkey credentials: %w", err)
	}
	return count, nil
}

func (r *PasskeyRepository) UpdateCredentialAfterLogin(ctx context.Context, credentialID []byte, signCount uint32, cloneWarning, backupState bool, lastUsedAt time.Time) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE passkey_credentials
		SET sign_count = $2, clone_warning = $3, backup_state = $4, last_used_at = $5, updated_at = $5
		WHERE credential_id = $1 AND revoked_at IS NULL
	`, credentialID, signCount, cloneWarning, backupState, lastUsedAt)
	if err != nil {
		return fmt.Errorf("update passkey credential login: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *PasskeyRepository) RenameCredential(ctx context.Context, id, userID, name string, updatedAt time.Time) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE passkey_credentials
		SET name = $3, updated_at = $4
		WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL
	`, id, userID, name, updatedAt)
	if err != nil {
		return fmt.Errorf("rename passkey credential: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *PasskeyRepository) RevokeCredential(ctx context.Context, id, userID string, revokedAt time.Time) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE passkey_credentials
		SET revoked_at = $3, updated_at = $3
		WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL
	`, id, userID, revokedAt)
	if err != nil {
		return fmt.Errorf("revoke passkey credential: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *PasskeyRepository) CreateChallenge(ctx context.Context, challenge *models.WebAuthnChallenge) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO webauthn_challenges (id, user_id, ceremony, challenge, session_data, expires_at, used_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, challenge.ID, challenge.UserID, challenge.Ceremony, challenge.Challenge, challenge.SessionData,
		challenge.ExpiresAt, challenge.UsedAt, challenge.CreatedAt)
	if err != nil {
		return fmt.Errorf("create webauthn challenge: %w", mapConflict(err))
	}
	return nil
}

func (r *PasskeyRepository) ConsumeChallenge(ctx context.Context, ceremony, challenge string, userID *string, usedAt time.Time) (*models.WebAuthnChallenge, error) {
	query := `
		UPDATE webauthn_challenges
		SET used_at = $4
		WHERE ceremony = $1
			AND challenge = $2
			AND used_at IS NULL
			AND expires_at > $4
			AND (($3::uuid IS NULL AND user_id IS NULL) OR user_id = $3::uuid)
		RETURNING id, user_id, ceremony, challenge, session_data, expires_at, used_at, created_at
	`
	record, err := scanWebAuthnChallenge(r.pool.QueryRow(ctx, query, ceremony, challenge, userID, usedAt))
	if err != nil {
		return nil, fmt.Errorf("consume webauthn challenge: %w", mapNotFound(err))
	}
	return record, nil
}

func scanPasskeyCredential(row interface {
	Scan(dest ...any) error
},
) (*models.PasskeyCredential, error) {
	var credential models.PasskeyCredential
	var signCount int64
	if err := row.Scan(
		&credential.ID,
		&credential.UserID,
		&credential.CredentialID,
		&credential.PublicKey,
		&credential.AttestationType,
		&credential.Transports,
		&signCount,
		&credential.CloneWarning,
		&credential.BackupEligible,
		&credential.BackupState,
		&credential.Name,
		&credential.AAGUID,
		&credential.LastUsedAt,
		&credential.RevokedAt,
		&credential.CreatedAt,
		&credential.UpdatedAt,
	); err != nil {
		return nil, err
	}
	credential.SignCount = uint32(signCount)
	return &credential, nil
}

func scanWebAuthnChallenge(row interface {
	Scan(dest ...any) error
},
) (*models.WebAuthnChallenge, error) {
	var challenge models.WebAuthnChallenge
	if err := row.Scan(
		&challenge.ID,
		&challenge.UserID,
		&challenge.Ceremony,
		&challenge.Challenge,
		&challenge.SessionData,
		&challenge.ExpiresAt,
		&challenge.UsedAt,
		&challenge.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &challenge, nil
}

func mapConflict(err error) error {
	var pgErr *pgconn.PgError
	if err != nil && errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return repository.ErrConflict
	}
	return err
}
