-- Events table stores all ingested events.
-- tenant_id + event_id is primary key to enforce idempotency.
-- This ensures retries do not create duplicate events.

CREATE TABLE IF NOT EXISTS events (
  tenant_id   TEXT        NOT NULL,
  event_id    TEXT        NOT NULL,
  event_name  TEXT        NOT NULL,
  ts          TIMESTAMPTZ NOT NULL,
  properties  JSONB       NOT NULL DEFAULT '{}'::jsonb,
  ingested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, event_id)
);

-- Composite index optimized for metrics queries:
-- WHERE tenant_id=? AND event_name=? AND ts BETWEEN ...
CREATE INDEX IF NOT EXISTS idx_events_tenant_name_ts
  ON events(tenant_id, event_name, ts);
