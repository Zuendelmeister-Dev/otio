package main

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"os"
	"time"
)

var db *sql.DB

func getenv(name string, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

func openDB() error {
	dsn := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=disable", getenv("POSTGRES_HOST", "postgres"), getenv("POSTGRES_PORT", "5432"), getenv("POSTGRES_DB", "iotdb"), getenv("POSTGRES_USER", "iot"), getenv("POSTGRES_PASSWORD", "iotpass"))
	var err error
	db, err = sql.Open("postgres", getenv("POSTGRES_DSN", dsn))
	return err
}

func ensureSchema() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Serialize schema changes even if a predecessor's database backend is
	// still finishing after its application process has exited.
	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(1869900143)"); err != nil {
		return err
	}
	statements := []string{
		`CREATE TABLE IF NOT EXISTS metric_events (id BIGSERIAL PRIMARY KEY, ts TIMESTAMPTZ NOT NULL, agent_id TEXT NOT NULL, topic TEXT NOT NULL, metric_name TEXT NOT NULL, metric_value DOUBLE PRECISION, metric_text TEXT, metric_unit TEXT, metric_type TEXT, source_type TEXT, source_host TEXT, source_address TEXT, quality_status TEXT, payload JSONB NOT NULL)`,
		`ALTER TABLE metric_events ALTER COLUMN metric_value DROP NOT NULL`,
		`ALTER TABLE metric_events ADD COLUMN IF NOT EXISTS metric_text TEXT`,
		`CREATE INDEX IF NOT EXISTS idx_metric_events_agent_ts ON metric_events (agent_id, ts DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_metric_events_metric_ts ON metric_events (agent_id, metric_name, ts DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_metric_events_topic_ts ON metric_events (topic, ts DESC)`,
		`CREATE TABLE IF NOT EXISTS agent_status (agent_id TEXT PRIMARY KEY, ts TIMESTAMPTZ NOT NULL, connected BOOLEAN NOT NULL, healthy BOOLEAN NOT NULL, source_type TEXT, source_host TEXT, payload JSONB NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS error_events (id BIGSERIAL PRIMARY KEY, ts TIMESTAMPTZ NOT NULL, agent_id TEXT, severity TEXT, message TEXT NOT NULL, payload JSONB NOT NULL)`,
		`CREATE INDEX IF NOT EXISTS idx_error_events_agent_ts ON error_events (agent_id, ts DESC)`,
	}
	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return tx.Commit()
}
