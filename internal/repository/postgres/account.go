package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"
)

type accountRepository struct {
	BaseRepository
}

func NewAccountRepository(base BaseRepository) repository.AccountRepository {
	return &accountRepository{base}
}

// All account repository methods here

func (r *accountRepository) Create(ctx context.Context, account *model.Account) error {
	query := `
		INSERT INTO accounts (
			id, name, email, password_hash, status,
			created_at, updated_at, region_code
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	account.ID = uuid.New()
	account.CreatedAt = time.Now()
	account.UpdatedAt = time.Now()

	return r.WithTx(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, query,
			account.ID,
			account.Name,
			account.Email,
			account.PasswordHash,
			account.Status,
			account.CreatedAt,
			account.UpdatedAt,
			r.GetRegionFromContext(ctx),
		)
		return err
	})
}

func (r *accountRepository) Get(ctx context.Context, id uuid.UUID) (*model.Account, error) {
	query := `
		SELECT id, name, email, status, created_at, updated_at
		FROM accounts
		WHERE id = $1
	`
	var account model.Account
	err := r.db.GetContext(ctx, &account, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	return &account, nil
}

func (r *accountRepository) Update(ctx context.Context, account *model.Account) error {
	query := `
		UPDATE accounts
		SET name = $1, email = $2, status = $3, updated_at = $4
		WHERE id = $5
	`
	account.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, query,
		account.Name,
		account.Email,
		account.Status,
		account.UpdatedAt,
		account.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update account: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("account not found")
	}

	return nil
}

func (r *accountRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM accounts
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("account not found")
	}

	return nil
}

func (r *accountRepository) List(ctx context.Context) ([]*model.Account, error) {
	query := `
		SELECT id, name, email, status, created_at, updated_at
		FROM accounts
		ORDER BY created_at DESC
	`
	var accounts []*model.Account
	err := r.db.SelectContext(ctx, &accounts, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	return accounts, nil
}

// ... rest of account methods
