package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const passwordHashVersion = 19

type PasswordParams struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

func DefaultPasswordParams() PasswordParams {
	return PasswordParams{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

func HashPassword(password string, params PasswordParams) (string, error) {
	if password == "" {
		slog.Warn("password hashing rejected empty password")
		return "", fmt.Errorf("password is empty")
	}

	slog.Debug("password hashing started", "memory", params.Memory, "iterations", params.Iterations, "parallelism", params.Parallelism)
	salt := make([]byte, params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		slog.Error("password salt generation failed", "error", err)
		return "", fmt.Errorf("generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, params.Iterations, params.Memory, params.Parallelism, params.KeyLength)
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)

	slog.Debug("password hashing completed", "memory", params.Memory, "iterations", params.Iterations, "parallelism", params.Parallelism)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", passwordHashVersion, params.Memory, params.Iterations, params.Parallelism, encodedSalt, encodedHash), nil
}

func VerifyPassword(password, encodedHash string) (bool, error) {
	params, salt, hash, err := decodePasswordHash(encodedHash)
	if err != nil {
		slog.Warn("password verification rejected malformed hash", "error", err)
		return false, err
	}

	keyLength, err := intToUint32(len(hash))
	if err != nil {
		return false, err
	}
	otherHash := argon2.IDKey([]byte(password), salt, params.Iterations, params.Memory, params.Parallelism, keyLength)
	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		slog.Debug("password verification completed", "success", true)
		return true, nil
	}
	slog.Debug("password verification completed", "success", false)
	return false, nil
}

func decodePasswordHash(encodedHash string) (params PasswordParams, salt, hash []byte, err error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return params, nil, nil, fmt.Errorf("invalid password hash format")
	}
	if parts[2] != "v=19" {
		return params, nil, nil, fmt.Errorf("unsupported argon2 version")
	}

	params = PasswordParams{}
	for item := range strings.SplitSeq(parts[3], ",") {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			return params, nil, nil, fmt.Errorf("invalid argon2 parameters")
		}
		parsed, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return params, nil, nil, fmt.Errorf("invalid argon2 parameter %s: %w", key, err)
		}
		switch key {
		case "m":
			params.Memory = uint32(parsed)
		case "t":
			params.Iterations = uint32(parsed)
		case "p":
			if parsed > 255 {
				return params, nil, nil, fmt.Errorf("argon2 parallelism is too large: %d", parsed)
			}
			params.Parallelism = uint8(parsed)
		default:
			return params, nil, nil, fmt.Errorf("unknown argon2 parameter: %s", key)
		}
	}

	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return params, nil, nil, fmt.Errorf("decode salt: %w", err)
	}
	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return params, nil, nil, fmt.Errorf("decode hash: %w", err)
	}
	params.SaltLength, err = intToUint32(len(salt))
	if err != nil {
		return params, nil, nil, err
	}
	params.KeyLength, err = intToUint32(len(hash))
	if err != nil {
		return params, nil, nil, err
	}

	return params, salt, hash, nil
}

func intToUint32(value int) (uint32, error) {
	if value < 0 || value > int(^uint32(0)) {
		return 0, fmt.Errorf("value does not fit uint32: %d", value)
	}
	return uint32(value), nil
}
