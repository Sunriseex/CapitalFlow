package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/sunriseex/capitalflow/internal/models"
)

type AuditEventRepository struct {
	pool queryExecer
}

func NewAuditEventRepository(pool queryExecer) *AuditEventRepository {
	return &AuditEventRepository{pool: pool}
}

func (r *AuditEventRepository) Create(ctx context.Context, event *models.AuditEvent) error {
	if err := insertAuditEvent(ctx, r.pool, event); err != nil {
		return fmt.Errorf("create audit event: %w", err)
	}
	return nil
}

func newAuditEvent(actorUserID *string, eventType, entityType, entityID string, afterSummary any) (*models.AuditEvent, error) {
	after, err := json.Marshal(afterSummary)
	if err != nil {
		return nil, fmt.Errorf("encode audit after summary: %w", err)
	}
	return &models.AuditEvent{
		ID:           uuid.NewString(),
		ActorUserID:  actorUserID,
		EventType:    eventType,
		EntityType:   entityType,
		EntityID:     entityID,
		AfterSummary: after,
		CreatedAt:    time.Now().UTC(),
	}, nil
}

func insertAuditEvent(ctx context.Context, execer sqlExecer, event *models.AuditEvent) error {
	_, err := execer.Exec(ctx, `
		INSERT INTO audit_events (
			id, actor_user_id, event_type, entity_type, entity_id,
			before_summary, after_summary, ip_address, user_agent, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7::jsonb, $8::inet, $9, $10)
	`, event.ID, event.ActorUserID, event.EventType, event.EntityType, event.EntityID,
		nullableJSON(event.BeforeSummary), nullableJSON(event.AfterSummary), event.IPAddress, event.UserAgent, event.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}

func nullableJSON(value json.RawMessage) any {
	if len(value) == 0 {
		return nil
	}
	return string(value)
}
