package models

import (
	"encoding/json"
	"time"
)

type AuditEvent struct {
	ID            string          `json:"id"`
	ActorUserID   *string         `json:"actor_user_id,omitempty"`
	EventType     string          `json:"event_type"`
	EntityType    string          `json:"entity_type"`
	EntityID      string          `json:"entity_id"`
	BeforeSummary json.RawMessage `json:"before_summary,omitempty"`
	AfterSummary  json.RawMessage `json:"after_summary,omitempty"`
	IPAddress     *string         `json:"ip_address,omitempty"`
	UserAgent     string          `json:"user_agent,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}
