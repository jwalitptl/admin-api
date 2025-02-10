package postgres

import (
	"context"

	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"
)

type notificationRepository struct {
	*BaseRepository
}

func NewNotificationRepository(base BaseRepository) repository.NotificationRepository {
	return &notificationRepository{
		BaseRepository: &base,
	}
}

func (r *notificationRepository) Create(ctx context.Context, notification *model.Notification) error {
	query := `INSERT INTO notifications (user_id, organization_id, channel, recipient, subject, content, status, created_at, updated_at) 
             VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`
	return r.GetDB().QueryRowContext(ctx, query,
		notification.UserID, notification.OrganizationID, notification.Channel,
		notification.Recipient, notification.Subject, notification.Content,
		notification.Status, notification.CreatedAt, notification.UpdatedAt).Scan(&notification.ID)
}

func (r *notificationRepository) Update(ctx context.Context, notification *model.Notification) error {
	query := `UPDATE notifications SET status = $1, updated_at = $2, last_error = $3, retry_count = $4, next_retry_at = $5 WHERE id = $6`
	_, err := r.GetDB().ExecContext(ctx, query,
		notification.Status, notification.UpdatedAt, notification.LastError,
		notification.RetryCount, notification.NextRetryAt, notification.ID)
	return err
}
