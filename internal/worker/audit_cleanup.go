package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/jwalitptl/admin-api/internal/repository"
)

type AuditCleanupWorker struct {
	repo            repository.AuditRepository
	retentionDays   int
	cleanupInterval time.Duration
}

func NewAuditCleanupWorker(repo repository.AuditRepository, retentionDays int, cleanupInterval time.Duration) *AuditCleanupWorker {
	return &AuditCleanupWorker{
		repo:            repo,
		retentionDays:   retentionDays,
		cleanupInterval: cleanupInterval,
	}
}

func (w *AuditCleanupWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.cleanup(ctx); err != nil {
				// Log error but continue
				fmt.Printf("Error cleaning up audit logs: %v\n", err)
			}
		}
	}
}

func (w *AuditCleanupWorker) cleanup(ctx context.Context) error {
	cutoff := time.Now().AddDate(0, 0, -w.retentionDays)

	rows, err := w.repo.Cleanup(ctx, cutoff)
	if err != nil {
		return fmt.Errorf("failed to cleanup audit logs: %w", err)
	}

	fmt.Printf("Cleaned up %d audit logs older than %v\n", rows, cutoff)
	return nil
}
