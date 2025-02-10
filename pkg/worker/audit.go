package worker

import (
	"context"
	"time"

	"github.com/jwalitptl/admin-api/internal/repository"
)

type AuditCleanupWorker struct {
	repo          repository.AuditRepository
	retentionDays int
	interval      time.Duration
}

func NewAuditCleanupWorker(repo repository.AuditRepository, retentionDays int, interval time.Duration) *AuditCleanupWorker {
	return &AuditCleanupWorker{
		repo:          repo,
		retentionDays: retentionDays,
		interval:      interval,
	}
}

func (w *AuditCleanupWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().AddDate(0, 0, -w.retentionDays)
			w.repo.DeleteBefore(ctx, cutoff)
		}
	}
}
