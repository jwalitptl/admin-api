package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jwalitptl/admin-api/internal/email"
	"github.com/jwalitptl/admin-api/internal/model"
	"github.com/jwalitptl/admin-api/internal/repository"
	"github.com/jwalitptl/admin-api/internal/service/audit"
	"github.com/jwalitptl/admin-api/pkg/messaging"
)

const (
	maxRetries = 3
	retryDelay = 5 * time.Second

	channelEmail = "email"
	channelSMS   = "sms"
	channelPush  = "push"
	channelInApp = "in_app"
)

type Service interface {
	Send(ctx context.Context, notification *model.Notification) error
}

type service struct {
	repo     repository.NotificationRepository
	emailSvc email.Service
	broker   messaging.Broker
	auditor  *audit.Service
}

func NewService(repo repository.NotificationRepository, emailSvc email.Service, broker messaging.Broker, auditor *audit.Service) Service {
	return &service{
		repo:     repo,
		emailSvc: emailSvc,
		broker:   broker,
		auditor:  auditor,
	}
}

func (s *service) Send(ctx context.Context, notification *model.Notification) error {
	if err := s.validateNotification(notification); err != nil {
		return fmt.Errorf("invalid notification: %w", err)
	}

	notification.ID = uuid.New()
	notification.CreatedAt = time.Now()
	notification.UpdatedAt = time.Now()
	notification.Status = model.NotificationStatusPending

	if err := s.repo.Create(ctx, notification); err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	s.auditor.Log(ctx, notification.UserID, notification.OrganizationID, "create", "notification", notification.ID, &audit.LogOptions{
		Changes: notification,
	})

	// Process notification asynchronously
	go s.processNotification(ctx, notification)

	return nil
}

func (s *service) processNotification(ctx context.Context, notification *model.Notification) {
	if ctx == nil {
		ctx = context.Background()
	}
	var err error
	switch notification.Channel {
	case channelEmail:
		err = s.sendEmail(ctx, notification)
	case channelSMS:
		err = s.sendSMS(ctx, notification)
	case channelPush:
		err = s.sendPush(ctx, notification)
	case channelInApp:
		err = s.sendInApp(ctx, notification)
	default:
		err = fmt.Errorf("unsupported channel: %s", notification.Channel)
	}

	if err != nil {
		s.handleError(ctx, notification, err)
		return
	}

	notification.Status = model.NotificationStatusSent
	notification.SentAt = time.Now()
	notification.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, notification); err != nil {
		s.auditor.Log(ctx, notification.UserID, notification.OrganizationID, "update_failed", "notification", notification.ID, &audit.LogOptions{
			Metadata: map[string]interface{}{
				"error": err.Error(),
			},
		})
		return
	}

	s.auditor.Log(ctx, notification.UserID, notification.OrganizationID, "sent", "notification", notification.ID, &audit.LogOptions{
		Metadata: map[string]interface{}{
			"channel": notification.Channel,
			"sent_at": notification.SentAt,
		},
	})
}

func (s *service) sendEmail(ctx context.Context, notification *model.Notification) error {
	return s.emailSvc.SendCustom(ctx, notification.Recipient, notification.Subject, notification.Content)
}

func (s *service) sendSMS(_ context.Context, _ *model.Notification) error {
	// Implement SMS sending logic
	return fmt.Errorf("SMS sending not implemented")
}

func (s *service) sendPush(_ context.Context, _ *model.Notification) error {
	// Implement push notification logic
	return fmt.Errorf("push notifications not implemented")
}

func (s *service) sendInApp(ctx context.Context, notification *model.Notification) error {
	event := &model.NotificationEvent{
		ID:             uuid.New(),
		NotificationID: notification.ID,
		UserID:         notification.UserID,
		Type:           "in_app_notification",
		Content:        notification.Content,
		CreatedAt:      time.Now(),
	}

	return s.broker.Publish(ctx, "notifications", event)
}

func (s *service) handleError(ctx context.Context, notification *model.Notification, err error) {
	notification.RetryCount++
	notification.LastError = err.Error()
	notification.Status = model.NotificationStatusFailed
	notification.UpdatedAt = time.Now()

	if notification.RetryCount >= maxRetries {
		notification.Status = model.NotificationStatusFailed
	} else {
		notification.Status = model.NotificationStatusRetrying
		notification.NextRetryAt = time.Now().Add(retryDelay * time.Duration(notification.RetryCount))
	}

	if err := s.repo.Update(ctx, notification); err != nil {
		s.auditor.Log(ctx, notification.UserID, notification.OrganizationID, "update_failed", "notification", notification.ID, &audit.LogOptions{
			Metadata: map[string]interface{}{
				"error": err.Error(),
			},
		})
		return
	}

	s.auditor.Log(ctx, notification.UserID, notification.OrganizationID, "send_failed", "notification", notification.ID, &audit.LogOptions{
		Metadata: map[string]interface{}{
			"error":       err.Error(),
			"retry_count": notification.RetryCount,
			"next_retry":  notification.NextRetryAt,
		},
	})
}

func (s *service) validateNotification(notification *model.Notification) error {
	if notification.UserID == uuid.Nil {
		return fmt.Errorf("user ID is required")
	}

	if notification.OrganizationID == uuid.Nil {
		return fmt.Errorf("organization ID is required")
	}

	if notification.Channel == "" {
		return fmt.Errorf("channel is required")
	}

	if notification.Recipient == "" {
		return fmt.Errorf("recipient is required")
	}

	if notification.Content == "" {
		return fmt.Errorf("content is required")
	}

	return nil
}
