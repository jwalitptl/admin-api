package postgres

import (
	"github.com/jwalitptl/admin-api/internal/repository"
)

type notificationRepository struct {
	*BaseRepository
}

func NewNotificationRepository(base *BaseRepository) repository.NotificationRepository {
	return &notificationRepository{
		BaseRepository: base,
	}
}
