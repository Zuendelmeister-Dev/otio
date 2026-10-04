package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/lib/pq"
)

type measurementFreshness struct {
	AgentID           string `json:"agentId"`
	MetricName        string `json:"metricName,omitempty"`
	LastMeasurementAt string `json:"lastMeasurementAt,omitempty"`
	AgeSeconds        *int64 `json:"ageSeconds"`
	Status            string `json:"status"`
	Warning           string `json:"warning,omitempty"`
}

func describeMeasurementFreshness(agent, metric string, timestamp sql.NullTime, now time.Time, staleAfter time.Duration) measurementFreshness {
	item := measurementFreshness{AgentID: agent, MetricName: metric, Status: "missing", Warning: "No measurements have been stored yet. Check the source, collector and MQTT connection."}
	if !timestamp.Valid {
		return item
	}
	item.LastMeasurementAt = scanTimeString(timestamp.Time)
	age := now.Sub(timestamp.Time)
	seconds := int64(age / time.Second)
	item.AgeSeconds = &seconds
	if age < -time.Minute {
		item.Status = "clock-skew"
		item.Warning = "The measurement timestamp is in the future. Check the source and collector clocks before treating this value as current."
		return item
	}
	if seconds < 0 {
		seconds = 0
	}
	item.Status = "fresh"
	item.Warning = ""
	if age >= staleAfter {
		item.Status = "stale"
		item.Warning = "No new measurements within the configured freshness window. The last stored value is historical; check the source, collector and MQTT connection."
	}
	return item
}

func apiHistorianStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	config, lastRun := historianSnapshot()
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	connected := db != nil && db.PingContext(ctx) == nil
	w.Header().Set("Content-Type", "application/json")
	warning := ""
	if !connected {
		warning = "Historian database is unavailable. Stored values and retention status cannot be verified; check PostgreSQL and the Lense connection settings."
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	writeJSON(w, map[string]any{
		"database":  map[string]any{"connected": connected, "warning": warning},
		"retention": map[string]any{"enabled": config.enabled(), "metricRetentionDays": config.MetricDays, "errorRetentionDays": config.ErrorDays, "intervalSeconds": int64(config.Interval / time.Second), "batchSize": config.BatchSize, "maxBatchesPerTable": config.MaxBatches, "lastRun": lastRun},
		"freshness": map[string]any{"staleAfterSeconds": int64(config.StaleAfter / time.Second), "timestampBasis": "measurement timestamp"},
	})
}

func apiHistorianFreshness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	config, _ := historianSnapshot()
	agent := strings.TrimSpace(r.URL.Query().Get("agent_id"))
	metric := strings.TrimSpace(r.URL.Query().Get("metric_name"))
	if len(agent) > 256 || len(metric) > 256 || (metric != "" && agent == "") {
		http.Error(w, "metric_name requires one configured agent_id; names must be at most 256 bytes", http.StatusBadRequest)
		return
	}
	ids := configuredAgentIDs()
	if agent != "" {
		if !isConfiguredAgent(agent) {
			http.Error(w, "agent is not in current topology", http.StatusNotFound)
			return
		}
		ids = []string{agent}
	}
	// Bound fan-out through the index. The UI may fetch individual agents as needed.
	truncated := len(ids) > 500
	if truncated {
		ids = ids[:500]
	}
	items := []measurementFreshness{}
	if len(ids) == 0 {
		writeJSON(w, map[string]any{"items": items, "truncated": false, "staleAfterSeconds": int64(config.StaleAfter / time.Second)})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if db == nil {
		historianFreshnessUnavailable(w, fmt.Errorf("historian database is unavailable"))
		return
	}
	query := `SELECT requested.agent_id, latest.ts FROM unnest($1::text[]) AS requested(agent_id) LEFT JOIN LATERAL (SELECT ts FROM metric_events WHERE agent_id=requested.agent_id`
	args := []any{pq.Array(ids)}
	if metric != "" {
		query += ` AND metric_name=$2`
		args = append(args, metric)
	}
	query += ` ORDER BY ts DESC LIMIT 1) latest ON TRUE`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		historianFreshnessUnavailable(w, err)
		return
	}
	defer rows.Close()
	now := time.Now().UTC()
	for rows.Next() {
		var id string
		var timestamp sql.NullTime
		if err := rows.Scan(&id, &timestamp); err != nil {
			historianFreshnessUnavailable(w, err)
			return
		}
		items = append(items, describeMeasurementFreshness(id, metric, timestamp, now, config.StaleAfter))
	}
	if err := rows.Err(); err != nil {
		historianFreshnessUnavailable(w, err)
		return
	}
	writeJSON(w, map[string]any{"items": items, "truncated": truncated, "staleAfterSeconds": int64(config.StaleAfter / time.Second), "timestampBasis": "measurement timestamp"})
}

func historianFreshnessUnavailable(w http.ResponseWriter, err error) {
	log.Printf("Historian freshness query failed: %v", err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)
	writeJSON(w, map[string]any{"items": []measurementFreshness{}, "status": "unknown", "warning": "Measurement freshness cannot be checked because the historian query failed. Check PostgreSQL connectivity and Lense service logs."})
}
