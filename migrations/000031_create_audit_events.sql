-- +goose Up
CREATE TABLE audit_events (
    id UUID PRIMARY KEY,
    actor_user_id UUID,
    event_type TEXT NOT NULL CHECK (btrim(event_type) <> ''),
    entity_type TEXT NOT NULL CHECK (btrim(entity_type) <> ''),
    entity_id TEXT NOT NULL CHECK (btrim(entity_id) <> ''),
    before_summary JSONB,
    after_summary JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT audit_events_before_summary_object_check CHECK (
        before_summary IS NULL OR jsonb_typeof(before_summary) = 'object'
    ),
    CONSTRAINT audit_events_after_summary_object_check CHECK (
        after_summary IS NULL OR jsonb_typeof(after_summary) = 'object'
    )
);

CREATE INDEX audit_events_actor_created_at_idx ON audit_events (actor_user_id, created_at DESC);
CREATE INDEX audit_events_entity_created_at_idx ON audit_events (entity_type, entity_id, created_at DESC);
CREATE INDEX audit_events_event_type_created_at_idx ON audit_events (event_type, created_at DESC);

-- +goose StatementBegin
CREATE FUNCTION prevent_audit_event_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'audit events are immutable' USING ERRCODE = '55000';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER audit_events_immutable
    BEFORE UPDATE OR DELETE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION prevent_audit_event_mutation();

-- +goose Down
DROP TRIGGER IF EXISTS audit_events_immutable ON audit_events;
DROP FUNCTION IF EXISTS prevent_audit_event_mutation();
DROP TABLE IF EXISTS audit_events;
