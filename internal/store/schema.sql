-- Durable event storage.
-- Idempotency is enforced via PRIMARY KEY (tenant_id, event_id).

CREATE TABLE IF NOT EXISTS events (
  tenant_id   TEXT        NOT NULL,
  event_id    TEXT        NOT NULL,
  event_name  TEXT        NOT NULL,
  ts          TIMESTAMPTZ NOT NULL,
  properties  JSONB       NOT NULL DEFAULT '{}'::jsonb,
  ingested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, event_id)
);

-- Index optimized for metrics queries:
-- WHERE tenant_id=? AND event_name=? AND ts in [from,to)
CREATE INDEX IF NOT EXISTS idx_events_tenant_name_ts
  ON events(tenant_id, event_name, ts);
