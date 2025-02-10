package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	UserID         uuid.UUID       `json:"user_id" db:"user_id"`
	OrganizationID uuid.UUID       `json:"organization_id" db:"organization_id"`
	Action         string          `json:"action" db:"action"`
	EntityType     string          `json:"entity_type" db:"entity_type"`
	EntityID       uuid.UUID       `json:"entity_id" db:"entity_id"`
	Changes        json.RawMessage `json:"changes" db:"changes"`
	Metadata       json.RawMessage `json:"metadata" db:"metadata"`
	IPAddress      string          `json:"ip_address" db:"ip_address"`
	UserAgent      string          `json:"user_agent" db:"user_agent"`
	AccessReason   string          `json:"access_reason" db:"access_reason"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
}

const (
	// Action types
	AuditActionCreate = "create"
	AuditActionRead   = "read"
	AuditActionUpdate = "update"
	AuditActionDelete = "delete"
	AuditActionLogin  = "login"
	AuditActionLogout = "logout"

	// Entity types
	AuditEntityUser          = "user"
	AuditEntityPatient       = "patient"
	AuditEntityMedicalRecord = "medical_record"
	AuditEntityAppointment   = "appointment"
	AuditEntityRole          = "role"
	AuditEntityPermission    = "permission"
)

type AggregateResponse struct {
	TotalLogs      int64             `json:"total_logs"`
	ActionCounts   map[string]int    `json:"action_counts"`
	EntityCounts   map[string]int    `json:"entity_counts"`
	UserActivity   map[string]int    `json:"user_activity"`
	HourlyActivity map[int]int       `json:"hourly_activity"`
	TopIPs         []IPActivityCount `json:"top_ips"`
}

type IPActivityCount struct {
	IPAddress string `json:"ip_address"`
	Count     int    `json:"count"`
}

type AggregateStats struct {
	TotalLogs      int64             `json:"total_logs"`
	ActionCounts   map[string]int    `json:"action_counts"`
	EntityCounts   map[string]int    `json:"entity_counts"`
	UserActivity   map[string]int    `json:"user_activity"`
	HourlyActivity map[int]int       `json:"hourly_activity"`
	TopIPs         []IPActivityCount `json:"top_ips"`
}
