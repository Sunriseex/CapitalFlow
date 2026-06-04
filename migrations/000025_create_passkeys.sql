-- +goose Up
CREATE TABLE passkey_credentials (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id BYTEA NOT NULL,
    public_key BYTEA NOT NULL,
    attestation_type TEXT NOT NULL DEFAULT '',
    transports TEXT[] NOT NULL DEFAULT '{}',
    sign_count BIGINT NOT NULL DEFAULT 0 CHECK (sign_count >= 0),
    clone_warning BOOLEAN NOT NULL DEFAULT false,
    backup_eligible BOOLEAN NOT NULL DEFAULT false,
    backup_state BOOLEAN NOT NULL DEFAULT false,
    name TEXT NOT NULL,
    aaguid UUID,
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX passkey_credentials_credential_id_idx ON passkey_credentials (credential_id);
CREATE INDEX passkey_credentials_user_id_idx ON passkey_credentials (user_id, created_at DESC);
CREATE INDEX passkey_credentials_active_user_idx ON passkey_credentials (user_id, created_at DESC) WHERE revoked_at IS NULL;

CREATE TABLE webauthn_challenges (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    ceremony TEXT NOT NULL CHECK (ceremony IN ('registration', 'login')),
    challenge TEXT NOT NULL,
    session_data JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX webauthn_challenges_challenge_idx ON webauthn_challenges (challenge);
CREATE INDEX webauthn_challenges_lookup_idx ON webauthn_challenges (challenge, ceremony, expires_at) WHERE used_at IS NULL;
CREATE INDEX webauthn_challenges_user_idx ON webauthn_challenges (user_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS webauthn_challenges;
DROP TABLE IF EXISTS passkey_credentials;
