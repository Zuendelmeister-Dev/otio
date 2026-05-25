CREATE TABLE IF NOT EXISTS metric_events (
  id BIGSERIAL PRIMARY KEY,
  ts TIMESTAMPTZ NOT NULL,
  agent_id TEXT NOT NULL,
  topic TEXT NOT NULL,
  metric_name TEXT NOT NULL,
  metric_value DOUBLE PRECISION,
  metric_text TEXT,
  metric_unit TEXT,
  metric_type TEXT,
  source_type TEXT,
  source_host TEXT,
  source_address TEXT,
  quality_status TEXT,
  payload JSONB NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_metric_events_agent_ts ON metric_events (agent_id, ts DESC);
CREATE INDEX IF NOT EXISTS idx_metric_events_metric_ts ON metric_events (agent_id, metric_name, ts DESC);
CREATE INDEX IF NOT EXISTS idx_metric_events_topic_ts ON metric_events (topic, ts DESC);

CREATE TABLE IF NOT EXISTS agent_status (
  agent_id TEXT PRIMARY KEY,
  ts TIMESTAMPTZ NOT NULL,
  connected BOOLEAN NOT NULL,
  healthy BOOLEAN NOT NULL,
  source_type TEXT,
  source_host TEXT,
  payload JSONB NOT NULL
);

CREATE TABLE IF NOT EXISTS error_events (
  id BIGSERIAL PRIMARY KEY,
  ts TIMESTAMPTZ NOT NULL,
  agent_id TEXT,
  severity TEXT,
  message TEXT NOT NULL,
  payload JSONB NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_error_events_agent_ts ON error_events (agent_id, ts DESC);
