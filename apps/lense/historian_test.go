package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

type historianTestConnector struct{ connection *historianTestConnection }

func (c historianTestConnector) Connect(context.Context) (driver.Conn, error) {
	return c.connection, nil
}
func (c historianTestConnector) Driver() driver.Driver { return historianTestDriver{} }

type historianTestDriver struct{}

func (historianTestDriver) Open(string) (driver.Conn, error) { return nil, driver.ErrSkip }

type historianTestConnection struct {
	locked                                      bool
	deletes                                     []int64
	queries                                     []string
	arguments                                   [][]driver.NamedValue
	commits, rollbacks                          int
	failDelete, failQuery, failCommit, failPing bool
	rows                                        [][]driver.Value
	rowError                                    error
}

func (c *historianTestConnection) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (c *historianTestConnection) Close() error                        { return nil }
func (c *historianTestConnection) Begin() (driver.Tx, error)           { return c, nil }
func (c *historianTestConnection) BeginTx(ctx context.Context, _ driver.TxOptions) (driver.Tx, error) {
	if _, ok := ctx.Deadline(); !ok {
		return nil, errors.New("retention has no deadline")
	}
	return c, nil
}
func (c *historianTestConnection) Commit() error {
	if c.failCommit {
		return errors.New("commit failed")
	}
	c.commits++
	return nil
}
func (c *historianTestConnection) Rollback() error { c.rollbacks++; return nil }
func (c *historianTestConnection) Ping(context.Context) error {
	if c.failPing {
		return errors.New("private database failure")
	}
	return nil
}
func (c *historianTestConnection) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if _, ok := ctx.Deadline(); !ok {
		return nil, errors.New("unbounded retention statement")
	}
	c.queries = append(c.queries, query)
	c.arguments = append(c.arguments, args)
	if strings.HasPrefix(query, "DELETE") {
		if c.failDelete {
			return nil, errors.New("delete failed")
		}
		if len(c.deletes) == 0 {
			return driver.RowsAffected(0), nil
		}
		n := c.deletes[0]
		c.deletes = c.deletes[1:]
		return driver.RowsAffected(n), nil
	}
	return driver.RowsAffected(0), nil
}
func (c *historianTestConnection) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if _, ok := ctx.Deadline(); !ok {
		return nil, errors.New("unbounded historian query")
	}
	c.queries = append(c.queries, query)
	c.arguments = append(c.arguments, args)
	if c.failQuery {
		return nil, errors.New("private database query failure")
	}
	if strings.Contains(query, "pg_try_advisory_xact_lock") {
		return &historianTestRows{columns: []string{"locked"}, values: [][]driver.Value{{c.locked}}}, nil
	}
	return &historianTestRows{columns: []string{"agent_id", "ts"}, values: c.rows, failure: c.rowError}, nil
}

type historianTestRows struct {
	columns []string
	values  [][]driver.Value
	failure error
	index   int
}

func (r *historianTestRows) Columns() []string { return r.columns }
func (r *historianTestRows) Close() error      { return nil }
func (r *historianTestRows) Next(values []driver.Value) error {
	if r.index >= len(r.values) {
		if r.failure != nil {
			return r.failure
		}
		return io.EOF
	}
	copy(values, r.values[r.index])
	r.index++
	return nil
}
func historianTestDatabase(t *testing.T, c *historianTestConnection) *sql.DB {
	t.Helper()
	database := sql.OpenDB(historianTestConnector{connection: c})
	database.SetMaxOpenConns(1)
	t.Cleanup(func() { database.Close() })
	return database
}

func TestHistorianConfigDefaultsAndValidation(t *testing.T) {
	for name, value := range map[string]string{"LENSE_RETENTION_METRIC_DAYS": "0", "LENSE_RETENTION_ERROR_DAYS": "0", "LENSE_RETENTION_INTERVAL_SECONDS": "3600", "LENSE_RETENTION_BATCH_SIZE": "1000", "LENSE_RETENTION_MAX_BATCHES": "5", "LENSE_STALE_AFTER_SECONDS": "300"} {
		t.Setenv(name, value)
	}
	config, err := loadHistorianConfig()
	if err != nil || config.enabled() || config.StaleAfter != 5*time.Minute {
		t.Fatalf("unexpected safe defaults: %+v %v", config, err)
	}
	for _, value := range []string{"-1", "0.5", "tomorrow", "36501", ""} {
		t.Run("invalid_days_"+value, func(t *testing.T) {
			t.Setenv("LENSE_RETENTION_METRIC_DAYS", value)
			if _, err := loadHistorianConfig(); err == nil {
				t.Fatal("invalid retention accepted")
			}
		})
	}
	t.Setenv("LENSE_RETENTION_METRIC_DAYS", "30")
	t.Setenv("LENSE_RETENTION_ERROR_DAYS", "90")
	t.Setenv("LENSE_STALE_AFTER_SECONDS", "600")
	config, err = loadHistorianConfig()
	if err != nil || config.MetricDays != 30 || config.ErrorDays != 90 || config.StaleAfter != 10*time.Minute {
		t.Fatalf("custom config: %+v %v", config, err)
	}
}

// Only an explicitly supplied disposable test database is used. All objects
// live in a unique schema and cleanup never touches public historian tables.
func TestHistorianPostgresIntegration(t *testing.T) {
	dsn := os.Getenv("LENSE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set LENSE_TEST_POSTGRES_DSN for a disposable PostgreSQL integration database")
	}
	database, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	defer database.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	schema := fmt.Sprintf("otio_historian_test_%d", time.Now().UnixNano())
	if _, err = database.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if _, err := database.ExecContext(cleanup, `DROP SCHEMA `+schema+` CASCADE`); err != nil {
			t.Errorf("clean test schema: %v", err)
		}
	}()
	if _, err = database.ExecContext(ctx, `SET search_path TO `+schema); err != nil {
		t.Fatal(err)
	}
	previous := db
	db = database
	defer func() { db = previous }()
	if err = ensureSchema(); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err = database.ExecContext(ctx, `INSERT INTO metric_events(ts,agent_id,topic,metric_name,payload) VALUES ($1,'source','demo/source/temperature','temperature','{}'),($2,'source','demo/source/temperature','temperature','{}')`, now.Add(-48*time.Hour), now); err != nil {
		t.Fatal(err)
	}
	if _, err = database.ExecContext(ctx, `INSERT INTO error_events(ts,message,payload) VALUES ($1,'old','{}'),($2,'recent','{}')`, now.Add(-48*time.Hour), now); err != nil {
		t.Fatal(err)
	}
	if _, err = database.ExecContext(ctx, `INSERT INTO agent_status(agent_id,ts,connected,healthy,payload) VALUES ('source',$1,true,true,'{}')`, now.Add(-48*time.Hour)); err != nil {
		t.Fatal(err)
	}
	config := defaultHistorianConfig()
	config.MetricDays = 1
	config.ErrorDays = 1
	config.BatchSize = 1
	run, err := runHistorianRetention(ctx, database, config, now)
	if err != nil || run.MetricsDeleted != 1 || run.ErrorsDeleted != 1 {
		t.Fatalf("real retention: %+v %v", run, err)
	}
	for _, table := range []string{"metric_events", "error_events", "agent_status"} {
		var count int
		if err := database.QueryRowContext(ctx, `SELECT count(*) FROM `+table).Scan(&count); err != nil || count != 1 {
			t.Fatalf("retention counts %s: %d %v", table, count, err)
		}
	}
	var last time.Time
	if err := database.QueryRowContext(ctx, `SELECT latest.ts FROM unnest($1::text[]) AS requested(agent_id) LEFT JOIN LATERAL (SELECT ts FROM metric_events WHERE agent_id=requested.agent_id AND metric_name=$2 ORDER BY ts DESC LIMIT 1) latest ON TRUE`, `{source}`, "temperature").Scan(&last); err != nil || describeMeasurementFreshness("source", "temperature", sql.NullTime{Time: last, Valid: true}, now, config.StaleAfter).Status != "fresh" {
		t.Fatalf("real indexed freshness: %v %v", last, err)
	}
}

func TestHistorianRetentionIsDisabledByDefault(t *testing.T) {
	run, err := runHistorianRetention(context.Background(), nil, defaultHistorianConfig(), time.Now())
	if err != nil || run.MetricsDeleted != 0 || run.ErrorsDeleted != 0 {
		t.Fatalf("disabled retention touched DB: %+v %v", run, err)
	}
}

func TestHistorianRetentionBoundsWorkAndPreservesStatus(t *testing.T) {
	c := &historianTestConnection{locked: true, deletes: []int64{2, 2, 1}}
	config := defaultHistorianConfig()
	config.MetricDays = 30
	config.ErrorDays = 90
	config.BatchSize = 2
	config.MaxBatches = 2
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	run, err := runHistorianRetention(context.Background(), historianTestDatabase(t, c), config, now)
	if err != nil || run.MetricsDeleted != 4 || run.ErrorsDeleted != 1 || c.commits != 3 {
		t.Fatalf("bounded run: %+v %+v %v", run, c, err)
	}
	deletes := 0
	for i, query := range c.queries {
		if strings.Contains(query, "agent_status") || strings.Contains(query, "TRUNCATE") {
			t.Fatal("retention deletes live state")
		}
		if !strings.HasPrefix(query, "DELETE") {
			continue
		}
		deletes++
		if !strings.Contains(query, "ts < $1") || !strings.Contains(query, "LIMIT $2") {
			t.Fatal("unbounded delete", query)
		}
		days := 30
		if strings.Contains(query, "error_events") {
			days = 90
		}
		if !c.arguments[i][0].Value.(time.Time).Equal(now.Add(-time.Duration(days)*24*time.Hour)) || c.arguments[i][1].Value != int64(2) {
			t.Fatal("wrong cutoff or batch limit", c.arguments[i])
		}
	}
	if deletes != 3 {
		t.Fatal("unexpected number of delete batches", deletes)
	}
}

func TestHistorianRetentionLocksAndRollsBackFailedBatch(t *testing.T) {
	for _, scenario := range []string{"locked_elsewhere", "delete_error", "query_error", "commit_error"} {
		t.Run(scenario, func(t *testing.T) {
			c := &historianTestConnection{locked: scenario != "locked_elsewhere", failDelete: scenario == "delete_error", failQuery: scenario == "query_error", failCommit: scenario == "commit_error", deletes: []int64{1}}
			config := defaultHistorianConfig()
			config.MetricDays = 30
			run, err := runHistorianRetention(context.Background(), historianTestDatabase(t, c), config, time.Now())
			if (err != nil) != (scenario != "locked_elsewhere") || run.Skipped != (scenario == "locked_elsewhere") || run.MetricsDeleted != 0 || c.commits != 0 {
				t.Fatalf("unsafe failed/locked batch: %+v %v %+v", run, err, c)
			}
			if scenario != "commit_error" && c.rollbacks != 1 {
				t.Fatal("failed batch not rolled back")
			}
		})
	}
}

func TestMeasurementFreshnessBoundaryAndClockSkew(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name   string
		age    time.Duration
		valid  bool
		status string
	}{
		{"missing", 0, false, "missing"}, {"fresh", 299 * time.Second, true, "fresh"}, {"boundary", 300 * time.Second, true, "stale"}, {"old", time.Hour, true, "stale"}, {"future", -2 * time.Minute, true, "clock-skew"}, {"small_clock_difference", -10 * time.Second, true, "fresh"},
	} {
		t.Run(test.name, func(t *testing.T) {
			item := describeMeasurementFreshness("source", "temperature", sql.NullTime{Time: now.Add(-test.age), Valid: test.valid}, now, 5*time.Minute)
			if item.Status != test.status || (item.Warning != "") != (test.status != "fresh") {
				t.Fatalf("freshness: %+v", item)
			}
			if !test.valid && item.AgeSeconds != nil {
				t.Fatal("missing data invented an age")
			}
		})
	}
}

func TestHistorianFreshnessQueriesOneMetricAndPropagatesFailures(t *testing.T) {
	previous := db
	t.Cleanup(func() { db = previous })
	for _, scenario := range []string{"success", "query_error", "rows_error", "scan_error"} {
		t.Run(scenario, func(t *testing.T) {
			c := &historianTestConnection{rows: [][]driver.Value{{"modbus-machine-01", time.Now().Add(-time.Hour)}}, failQuery: scenario == "query_error"}
			if scenario == "rows_error" {
				c.rowError = errors.New("private rows failure")
			}
			if scenario == "scan_error" {
				c.rows[0][1] = "invalid timestamp"
			}
			db = historianTestDatabase(t, c)
			w := httptest.NewRecorder()
			apiHistorianFreshness(w, httptest.NewRequest("GET", "/api/historian/freshness?agent_id=modbus-machine-01&metric_name=temperature", nil))
			if scenario != "success" {
				if w.Code != 503 || strings.Contains(w.Body.String(), "private") || !strings.Contains(w.Body.String(), `"unknown"`) {
					t.Fatalf("silent/private DB failure: %d %s", w.Code, w.Body.String())
				}
				return
			}
			var result struct{ Items []measurementFreshness }
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || w.Code != 200 || len(result.Items) != 1 || result.Items[0].Status != "stale" {
				t.Fatalf("freshness response: %d %s %v", w.Code, w.Body.String(), err)
			}
			if !strings.Contains(c.queries[0], "metric_name=$2") || !strings.Contains(c.queries[0], "LIMIT 1") || len(c.arguments[0]) != 2 || c.arguments[0][1].Value != "temperature" {
				t.Fatal("metric lookup not scoped/bounded", c.queries, c.arguments)
			}
		})
	}
}

func TestHistorianAPIRejectsMethodsAndUnavailableDatabase(t *testing.T) {
	previous := db
	db = nil
	t.Cleanup(func() { db = previous })
	w := httptest.NewRecorder()
	apiHistorianStatus(w, httptest.NewRequest("GET", "/api/historian/status", nil))
	if w.Code != 503 || !strings.Contains(w.Body.String(), `"connected":false`) {
		t.Fatalf("unavailable database: %d %s", w.Code, w.Body.String())
	}
	for _, handler := range []func(*httptest.ResponseRecorder){
		func(w *httptest.ResponseRecorder) {
			apiHistorianStatus(w, httptest.NewRequest("POST", "/api/historian/status", nil))
		},
		func(w *httptest.ResponseRecorder) {
			apiHistorianFreshness(w, httptest.NewRequest("DELETE", "/api/historian/freshness", nil))
		},
	} {
		w := httptest.NewRecorder()
		handler(w)
		if w.Code != 405 || w.Header().Get("Allow") != "GET" {
			t.Fatal("write method accepted")
		}
	}
	for _, request := range []struct {
		url  string
		code int
	}{{"/api/historian/freshness?metric_name=temperature", 400}, {"/api/historian/freshness?agent_id=unconfigured", 404}} {
		w := httptest.NewRecorder()
		apiHistorianFreshness(w, httptest.NewRequest("GET", request.url, nil))
		if w.Code != request.code {
			t.Fatalf("unscoped request: %d %s", w.Code, w.Body.String())
		}
	}
}
