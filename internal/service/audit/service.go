package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"
)

type Service struct {
	repo repository.AuditRepository
}

func NewService(repo repository.AuditRepository) *Service {
	return &Service{repo: repo}
}

type LogOptions struct {
	Changes     interface{}
	Metadata    interface{}
	IPAddress   string
	UserAgent   string
	AccessLevel string
}

// Log creates an audit log entry
func (s *Service) Log(ctx context.Context, userID, orgID uuid.UUID, action, entityType string, entityID uuid.UUID, opts *LogOptions) error {
	var changes, metadata json.RawMessage
	var err error

	if opts != nil {
		if opts.Changes != nil {
			changes, err = json.Marshal(opts.Changes)
			if err != nil {
				return err
			}
		}
		if opts.Metadata != nil {
			metadata, err = json.Marshal(opts.Metadata)
			if err != nil {
				return err
			}
		}
	}

	// Get IP and User Agent from gin context if not provided in opts
	ipAddress := opts.IPAddress
	userAgent := opts.UserAgent
	if gc, ok := ctx.(*gin.Context); ok && ipAddress == "" {
		ipAddress = gc.ClientIP()
		userAgent = gc.GetHeader("User-Agent")
	}

	log := &model.AuditLog{
		ID:             uuid.New(),
		UserID:         userID,
		OrganizationID: orgID,
		Action:         action,
		EntityType:     entityType,
		EntityID:       entityID,
		Changes:        changes,
		Metadata:       metadata,
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
		CreatedAt:      time.Now(),
	}

	return s.repo.Create(ctx, log)
}

func (s *Service) ListWithPagination(ctx context.Context, filters map[string]interface{}) ([]*model.AuditLog, int64, error) {
	return s.repo.ListWithPagination(ctx, filters)
}

func (s *Service) List(ctx context.Context, filters map[string]interface{}) ([]*model.AuditLog, error) {
	return s.repo.List(ctx, filters)
}

func (s *Service) GetAggregateStats(ctx context.Context, filters map[string]interface{}) (*model.AggregateStats, error) {
	return s.repo.GetAggregateStats(ctx, filters)
}

func (s *Service) LogEmergencyAccess(ctx context.Context, log *model.AuditLog) error {
	// Additional emergency logging logic
	log.EntityType = "emergency_access"
	return s.repo.CreateAuditLog(ctx, log)
}

func (s *Service) Cleanup(ctx context.Context, before time.Time) (int64, error) {
	return s.repo.Cleanup(ctx, before)
}
