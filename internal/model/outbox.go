package model

import (
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
	ID        uuid.UUID    `json:"id" db:"id"`
	EventType string       `json:"event_type" db:"event_type"`
	Payload   []byte       `json:"payload" db:"payload"`
	Status    OutboxStatus `json:"status" db:"status"`
	CreatedAt time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt time.Time    `json:"updated_at" db:"updated_at"`
	Error     *string      `json:"error" db:"error"`
}
