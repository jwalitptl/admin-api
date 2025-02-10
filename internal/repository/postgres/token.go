package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/jwalitptl/admin-api/internal/repository"
)

type tokenRepository struct {
	BaseRepository
}

func NewTokenRepository(base BaseRepository) repository.TokenRepository {
	return &tokenRepository{base}
}

func (r *tokenRepository) StoreResetToken(ctx context.Context, userID uuid.UUID, token string, expiry time.Time) error {
	return r.WithTx(ctx, func(tx *sqlx.Tx) error {
		query := `
			INSERT INTO user_tokens (user_id, token, type, expires_at, region_code, created_at)
			VALUES ($1, $2, 'reset', $3, $4, NOW())
			ON CONFLICT (user_id, type) DO UPDATE
			SET token = $2, expires_at = $3, updated_at = NOW()
		`
		_, err := tx.ExecContext(ctx, query, userID, token, expiry, r.GetRegionFromContext(ctx))
		return err
	})
}

func (r *tokenRepository) ValidateResetToken(ctx context.Context, token string) (uuid.UUID, error) {
	query := `
		SELECT user_id 
		FROM user_tokens 
		WHERE token = $1 
		AND type = 'reset'
		AND expires_at > NOW()
		AND used_at IS NULL
		AND region_code = $2
	`

	var userID uuid.UUID
	err := r.GetDB().GetContext(ctx, &userID, query, token, r.GetRegionFromContext(ctx))
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid or expired token")
	}

	return userID, nil
}

func (r *tokenRepository) InvalidateResetToken(ctx context.Context, token string) error {
	query := `
		UPDATE user_tokens 
		SET used_at = NOW() 
		WHERE token = $1 
		AND type = 'reset'
		AND region_code = $2
	`

	result, err := r.GetDB().ExecContext(ctx, query, token, r.GetRegionFromContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to invalidate token: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("token not found or already used")
	}

	return nil
}

func (r *tokenRepository) StoreVerificationToken(ctx context.Context, userID uuid.UUID, token string, expiry time.Time) error {
	query := `
		INSERT INTO user_tokens (user_id, token, type, expires_at, created_at)
		VALUES ($1, $2, 'verification', $3, NOW())
		ON CONFLICT (user_id, type) DO UPDATE
		SET token = $2, expires_at = $3, updated_at = NOW()
	`

	_, err := r.GetDB().ExecContext(ctx, query, userID, token, expiry)
	if err != nil {
		return fmt.Errorf("failed to store verification token: %w", err)
	}

	return nil
}

func (r *tokenRepository) ValidateVerificationToken(ctx context.Context, token string) (uuid.UUID, error) {
	query := `
		SELECT user_id 
		FROM user_tokens 
		WHERE token = $1 
		AND type = 'verification'
		AND expires_at > NOW()
		AND used_at IS NULL
	`

	var userID uuid.UUID
	err := r.GetDB().GetContext(ctx, &userID, query, token)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid or expired token")
	}

	return userID, nil
}

func (r *tokenRepository) InvalidateVerificationToken(ctx context.Context, token string) error {
	query := `
		UPDATE user_tokens 
		SET used_at = NOW() 
		WHERE token = $1 AND type = 'verification'
	`

	_, err := r.GetDB().ExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to invalidate token: %w", err)
	}

	return nil
}
