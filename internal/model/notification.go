package model

import (
	"time"

	"github.com/google/uuid"
)

type NotificationStatus string

const (
	NotificationStatusPending  NotificationStatus = "pending"
	NotificationStatusSent     NotificationStatus = "sent"
	NotificationStatusFailed   NotificationStatus = "failed"
	NotificationStatusRetrying NotificationStatus = "retrying"
)

type Notification struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	OrganizationID uuid.UUID
	Channel        string
	Priority       string
	Subject        string
	Content        string
	Recipient      string
	Status         NotificationStatus
	RetryCount     int
	LastError      string
	NextRetryAt    time.Time
	SentAt         time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type NotificationEvent struct {
	ID             uuid.UUID
	NotificationID uuid.UUID
	UserID         uuid.UUID
	Type           string
	Content        string
	CreatedAt      time.Time
}
