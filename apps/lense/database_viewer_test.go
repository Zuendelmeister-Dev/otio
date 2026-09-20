package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestViewerSelectGrammar(t *testing.T) {
	query, args, limit, err := parseViewerQuery(`SELECT ts, metric_value FROM public.metric_events WHERE agent_id = 'a''b' AND metric_value >= 12.5 ORDER BY ts DESC LIMIT 10;`)
	if err != nil || limit != 10 || len(args) != 2 || args[0] != "a'b" || strings.Contains(query, "a'b") || !strings.HasSuffix(query, "LIMIT 11") {
		t.Fatalf("%s %#v %d %v", query, args, limit, err)
	}
	for _, input := range []string{`SELECT * FROM agent_status`, `select "ts" from "public"."metric_events" where metric_text is not null limit 500`, `SELECT * FROM agent_status WHERE connected = true AND source_type ILIKE 'modbus%'`} {
		if _, _, _, err := parseViewerQuery(input); err != nil {
			t.Errorf("valid query: %s: %v", input, err)
		}
	}
	for _, input := range []string{`DELETE FROM metric_events`, `SELECT * FROM metric_events; DROP TABLE metric_events`, `SELECT pg_sleep(20) FROM metric_events`, `SELECT set_config('transaction_read_only','off',true) FROM metric_events`, `SELECT * INTO backup FROM metric_events`, `WITH x AS (DELETE FROM metric_events RETURNING *) SELECT * FROM x`, `SELECT * FROM pg_catalog.pg_authid`, `SELECT * FROM metric_events -- comment`, `SELECT * FROM metric_events UNION SELECT * FROM agent_status`, `SELECT * FROM metric_events FOR UPDATE`, `SELECT * FROM metric_events LIMIT 501`, `SELECT * FROM metric_events LIMIT 0`, `SELECT * FROM metric_events WHERE id = 1 OR true`, `SELECT * FROM metric_events WHERE id = (SELECT 1)`, `SELECT * FROM metric_events /*comment*/`, `SELECT * FROM metric_events WHERE id = 1; SELECT 2`} {
		if _, _, _, err := parseViewerQuery(input); err == nil {
			t.Errorf("unsafe/unsupported query accepted: %s", input)
		}
	}
}

type viewerTestDriver struct{}
type viewerTestConn struct{}
type viewerTestTx struct{}
type viewerTestRows struct{ n int }

func (viewerTestDriver) Open(string) (driver.Conn, error)  { return viewerTestConn{}, nil }
func (viewerTestConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (viewerTestConn) Close() error                        { return nil }
func (viewerTestConn) Begin() (driver.Tx, error)           { return viewerTestTx{}, nil }
func (viewerTestConn) BeginTx(_ context.Context, o driver.TxOptions) (driver.Tx, error) {
	if !o.ReadOnly {
		return nil, io.ErrUnexpectedEOF
	}
	return viewerTestTx{}, nil
}
func (viewerTestTx) Commit() error   { return nil }
func (viewerTestTx) Rollback() error { return nil }
func (viewerTestConn) ExecContext(_ context.Context, q string, _ []driver.NamedValue) (driver.Result, error) {
	if q != `SET LOCAL statement_timeout = '4s'` {
		return nil, io.ErrUnexpectedEOF
	}
	return driver.RowsAffected(0), nil
}
func (viewerTestConn) QueryContext(ctx context.Context, _ string, _ []driver.NamedValue) (driver.Rows, error) {
	if _, ok := ctx.Deadline(); !ok {
		return nil, io.ErrUnexpectedEOF
	}
	return &viewerTestRows{}, nil
}
func (*viewerTestRows) Columns() []string { return []string{"value", "nullable"} }
func (*viewerTestRows) Close() error      { return nil }
func (r *viewerTestRows) Next(dest []driver.Value) error {
	if r.n == 501 {
		return io.EOF
	}
	r.n++
	dest[0] = []byte("<script>not HTML</script>")
	dest[1] = nil
	return nil
}

func init() { sql.Register("viewer-test", viewerTestDriver{}) }
func TestViewerQueryIsBoundedAndReadOnly(t *testing.T) {
	previous := db
	testDB, err := sql.Open("viewer-test", "")
	if err != nil {
		t.Fatal(err)
	}
	db = testDB
	defer func() { db = previous; testDB.Close() }()
	w := httptest.NewRecorder()
	apiDatabaseQuery(w, httptest.NewRequest("POST", "/api/database/query", strings.NewReader(`{"sql":"SELECT * FROM metric_events LIMIT 2"}`)))
	var data struct {
		Rows      [][]any
		Truncated bool
		Limit     int
	}
	if err = json.Unmarshal(w.Body.Bytes(), &data); err != nil || w.Code != 200 || len(data.Rows) != 2 || !data.Truncated || data.Limit != 2 || data.Rows[0][1] != nil {
		t.Fatalf("%d %s %v", w.Code, w.Body.String(), err)
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("results must not be cached")
	}
	for _, body := range []string{`{"sql":"DELETE FROM metric_events"}`, `{"sql":"SELECT * FROM metric_events","unknown":true}`, `{"sql":"SELECT * FROM metric_events"} {}`} {
		w = httptest.NewRecorder()
		apiDatabaseQuery(w, httptest.NewRequest("POST", "/api/database/query", strings.NewReader(body)))
		if w.Code != 400 {
			t.Fatalf("accepted %s: %d", body, w.Code)
		}
	}
	w = httptest.NewRecorder()
	apiDatabaseQuery(w, httptest.NewRequest("GET", "/api/database/query", nil))
	if w.Code != 405 {
		t.Fatal(w.Code)
	}
}
