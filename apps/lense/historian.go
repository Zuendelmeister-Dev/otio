package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

// Retention is opt-in. Each table has its own age and a fixed work limit.
type historianConfig struct {
	MetricDays int           `json:"metricRetentionDays"`
	ErrorDays  int           `json:"errorRetentionDays"`
	Interval   time.Duration `json:"-"`
	BatchSize  int           `json:"batchSize"`
	MaxBatches int           `json:"maxBatchesPerTable"`
	StaleAfter time.Duration `json:"-"`
}

func defaultHistorianConfig() historianConfig {
	return historianConfig{Interval: time.Hour, BatchSize: 1000, MaxBatches: 5, StaleAfter: 5 * time.Minute}
}

func loadHistorianConfig() (historianConfig, error) {
	config := defaultHistorianConfig()
	fields := []struct {
		name     string
		min, max int
		set      func(int)
	}{
		{"LENSE_RETENTION_METRIC_DAYS", 0, 36500, func(v int) { config.MetricDays = v }},
		{"LENSE_RETENTION_ERROR_DAYS", 0, 36500, func(v int) { config.ErrorDays = v }},
		{"LENSE_RETENTION_INTERVAL_SECONDS", 300, 86400, func(v int) { config.Interval = time.Duration(v) * time.Second }},
		{"LENSE_RETENTION_BATCH_SIZE", 1, 10000, func(v int) { config.BatchSize = v }},
		{"LENSE_RETENTION_MAX_BATCHES", 1, 20, func(v int) { config.MaxBatches = v }},
		{"LENSE_STALE_AFTER_SECONDS", 30, 86400, func(v int) { config.StaleAfter = time.Duration(v) * time.Second }},
	}
	for _, field := range fields {
		raw, present := os.LookupEnv(field.name)
		if !present {
			continue
		}
		value, err := strconv.Atoi(raw)
		if err != nil || value < field.min || value > field.max {
			return historianConfig{}, fmt.Errorf("%s must be an integer between %d and %d", field.name, field.min, field.max)
		}
		field.set(value)
	}
	return config, nil
}

func (config historianConfig) enabled() bool { return config.MetricDays > 0 || config.ErrorDays > 0 }

type retentionRun struct {
	StartedAt      string `json:"startedAt"`
	FinishedAt     string `json:"finishedAt"`
	MetricsDeleted int64  `json:"metricsDeleted"`
	ErrorsDeleted  int64  `json:"errorsDeleted"`
	Skipped        bool   `json:"skipped"`
	Warning        string `json:"warning,omitempty"`
}

type historianManager struct {
	sync.Mutex
	config  historianConfig
	lastRun *retentionRun
}

var historian = &historianManager{config: defaultHistorianConfig()}

func startHistorianMaintenance(ctx context.Context, database *sql.DB, config historianConfig) {
	manager := &historianManager{config: config}
	historian = manager
	if !config.enabled() {
		return
	}
	go func() {
		ticker := time.NewTicker(config.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				run, err := runHistorianRetention(ctx, database, config, now.UTC())
				if err != nil {
					run.Warning = "Retention failed. The failed batch was rolled back; earlier completed batches remain applied. Check the Lense service logs."
					log.Printf("Historian retention failed: %v", err)
				}
				manager.Lock()
				manager.lastRun = &run
				manager.Unlock()
			}
		}
	}()
}

func historianSnapshot() (historianConfig, *retentionRun) {
	historian.Lock()
	defer historian.Unlock()
	var run *retentionRun
	if historian.lastRun != nil {
		copy := *historian.lastRun
		run = &copy
	}
	return historian.config, run
}

// A cycle stops after 30 seconds, even if the configured batch cap is larger.
// Transaction-scoped advisory locks serialize cleanup across Lense processes.
func runHistorianRetention(ctx context.Context, database *sql.DB, config historianConfig, now time.Time) (retentionRun, error) {
	run := retentionRun{StartedAt: scanTimeString(now)}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	finish := func(err error) (retentionRun, error) { run.FinishedAt = scanTimeString(time.Now()); return run, err }
	if !config.enabled() {
		return finish(nil)
	}
	if database == nil {
		return finish(fmt.Errorf("historian database is unavailable"))
	}
	if config.BatchSize < 1 || config.BatchSize > 10000 || config.MaxBatches < 1 || config.MaxBatches > 20 || config.MetricDays < 0 || config.MetricDays > 36500 || config.ErrorDays < 0 || config.ErrorDays > 36500 {
		return finish(fmt.Errorf("invalid retention configuration"))
	}
	for _, table := range []struct {
		name    string
		days    int
		deleted *int64
	}{
		{"metric_events", config.MetricDays, &run.MetricsDeleted},
		{"error_events", config.ErrorDays, &run.ErrorsDeleted},
	} {
		if table.days == 0 {
			continue
		}
		cutoff := now.Add(-time.Duration(table.days) * 24 * time.Hour)
		for batch := 0; batch < config.MaxBatches; batch++ {
			deleted, locked, err := pruneHistorianBatch(ctx, database, table.name, cutoff, config.BatchSize)
			if err != nil {
				return finish(err)
			}
			if !locked {
				run.Skipped = true
				return finish(nil)
			}
			*table.deleted += deleted
			if deleted < int64(config.BatchSize) {
				break
			}
		}
	}
	return finish(nil)
}

func pruneHistorianBatch(ctx context.Context, database *sql.DB, table string, cutoff time.Time, batchSize int) (int64, bool, error) {
	// Never let API input or configuration become a SQL table identifier.
	if table != "metric_events" && table != "error_events" {
		return 0, false, fmt.Errorf("unsupported retention table")
	}
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return 0, false, fmt.Errorf("begin retention batch: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SET LOCAL statement_timeout = '5s'`); err != nil {
		return 0, false, fmt.Errorf("set retention timeout: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `SET LOCAL lock_timeout = '1s'`); err != nil {
		return 0, false, fmt.Errorf("set retention lock timeout: %w", err)
	}
	var locked bool
	if err := tx.QueryRowContext(ctx, `SELECT pg_try_advisory_xact_lock(1869900144)`).Scan(&locked); err != nil {
		return 0, false, fmt.Errorf("lock retention batch: %w", err)
	}
	if !locked {
		return 0, false, nil
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE id IN (SELECT id FROM `+table+` WHERE ts < $1 ORDER BY ts, id LIMIT $2)`, cutoff, batchSize)
	if err != nil {
		return 0, true, fmt.Errorf("prune %s: %w", table, err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, true, fmt.Errorf("count pruned %s: %w", table, err)
	}
	if err := tx.Commit(); err != nil {
		return 0, true, fmt.Errorf("commit %s retention: %w", table, err)
	}
	return deleted, true, nil
}
