package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type OutboxStatus string

const (
	OutboxStatusPending   OutboxStatus = "PENDING"
	OutboxStatusProcessed OutboxStatus = "PROCESSED"
	OutboxStatusFailed    OutboxStatus = "FAILED"
)

type OutboxEvent struct {
	ID           uuid.UUID         `db:"id" json:"id"`
	EventType    string            `db:"event_type" json:"event_type"`
	Payload      json.RawMessage   `db:"payload" json:"payload"`
	Headers      map[string]string `json:"headers" db:"headers"`
	Status       string            `db:"status" json:"status"`
	ErrorMessage *string           `db:"error_message" json:"error_message,omitempty"`
	CreatedAt    time.Time         `db:"created_at" json:"created_at"`
	ProcessedAt  *time.Time        `db:"processed_at" json:"processed_at,omitempty"`
	UpdatedAt    time.Time         `db:"updated_at" json:"updated_at"`
	RetryCount   int               `db:"retry_count" json:"retry_count"`
	RetryAt      *time.Time        `db:"retry_at" json:"retry_at,omitempty"`
}
