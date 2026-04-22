package model

import (
	"time"

	"github.com/google/uuid"
)

// DisputeStatus represents the status of a dispute
type DisputeStatus string

const (
	DisputeStatusOpen          DisputeStatus = "OPEN"
	DisputeStatusInvestigating DisputeStatus = "INVESTIGATING"
	DisputeStatusResolved      DisputeStatus = "RESOLVED"
	DisputeStatusRejected      DisputeStatus = "REJECTED"
)

// Dispute represents a transaction dispute
type Dispute struct {
	ID         uuid.UUID     `json:"id"         db:"id"`
	TxID       string        `json:"txId"       db:"tx_id"`
	Reason     string        `json:"reason"     db:"reason"`
	Status     DisputeStatus `json:"status"     db:"status"`
	OpenedBy   string        `json:"openedBy"   db:"opened_by"`
	ResolvedBy *string       `json:"resolvedBy" db:"resolved_by"`
	Resolution *string       `json:"resolution" db:"resolution"`
	CreatedAt  time.Time     `json:"createdAt"  db:"created_at"`
	ResolvedAt *time.Time    `json:"resolvedAt" db:"resolved_at"`
}

// AuditLog represents an audit trail entry
type AuditLog struct {
	ID           uuid.UUID              `json:"id"           db:"id"`
	Action       string                 `json:"action"       db:"action"`
	Actor        string                 `json:"actor"        db:"actor"`
	ResourceType string                 `json:"resourceType" db:"resource_type"`
	ResourceID   *string                `json:"resourceId"   db:"resource_id"`
	Details      map[string]interface{} `json:"details"      db:"details"`
	IPAddress    *string                `json:"ipAddress"    db:"ip_address"`
	ChannelUsed  *string                `json:"channel"      db:"channel"`
	CreatedAt    time.Time              `json:"createdAt"    db:"created_at"`
}
