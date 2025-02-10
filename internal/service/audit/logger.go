package audit

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/jwalitptl/admin-api/internal/model"
)

type AuditLogger struct {
	service *Service
	mu      sync.Mutex
}

func NewAuditLogger(service *Service) *AuditLogger {
	return &AuditLogger{
		service: service,
	}
}

func (l *AuditLogger) Log(ctx context.Context, userID, orgID uuid.UUID, action, entityType string, entityID uuid.UUID, opts *LogOptions) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Async logging
	go func() {
		if err := l.service.Log(ctx, userID, orgID, action, entityType, entityID, opts); err != nil {
			// Handle error (maybe log to error monitoring service)
		}
	}()
}

func (l *AuditLogger) LogSync(ctx context.Context, userID, orgID uuid.UUID, action, entityType string, entityID uuid.UUID, opts *LogOptions) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.service.Log(ctx, userID, orgID, action, entityType, entityID, opts)
}

func (l *AuditLogger) LogEmergencyAccess(ctx context.Context, log *model.AuditLog) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Sync logging for emergency access
	if err := l.service.LogEmergencyAccess(ctx, log); err != nil {
		// Handle error
	}
}
