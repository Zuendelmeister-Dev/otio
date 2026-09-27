package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type historyDriver struct{}
type historyConn struct{}
type historyRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (historyDriver) Open(string) (driver.Conn, error)  { return historyConn{}, nil }
func (historyConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (historyConn) Close() error                        { return nil }
func (historyConn) Begin() (driver.Tx, error)           { return nil, driver.ErrSkip }
func (historyConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if strings.Contains(query, "FROM agent_status") {
		return &historyRows{columns: []string{"agent_id", "ts", "connected", "healthy", "source_type", "source_host", "payload"}}, nil
	}
	if len(args) != 1 || !strings.Contains(query, "agent_id=ANY($1)") {
		return nil, errors.New("source scope missing")
	}
	scope, _ := args[0].Value.(string)
	if !strings.Contains(scope, "modbus-machine-01") || strings.Contains(scope, "opcua-machine") {
		return nil, errors.New("wrong collector source scope")
	}
	if strings.Contains(query, "DISTINCT topic") {
		return &historyRows{columns: []string{"topic"}, values: [][]driver.Value{{"iot-lense/modbus-machine-01/metrics/temperature"}}}, nil
	}
	if !strings.Contains(query, "ORDER BY ts DESC LIMIT 50") {
		return nil, errors.New("history must be bounded")
	}
	return &historyRows{columns: []string{"ts", "topic", "payload"}, values: [][]driver.Value{{time.Now(), "iot-lense/modbus-machine-01/metrics/temperature", `{"value":23.5}`}}}, nil
}
func (r *historyRows) Columns() []string { return r.columns }
func (r *historyRows) Close() error      { return nil }
func (r *historyRows) Next(values []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(values, r.values[r.index])
	r.index++
	return nil
}
func init() { sql.Register("collector-history", historyDriver{}) }
func TestSenseHistoryUsesItsSources(t *testing.T) {
	previous := db
	testDB, err := sql.Open("collector-history", "")
	if err != nil {
		t.Fatal(err)
	}
	db = testDB
	defer func() { db = previous; testDB.Close() }()
	w := httptest.NewRecorder()
	apiAgent(w, httptest.NewRequest("GET", "/api/agents/iot-sense-modbus-01", nil))
	var result struct {
		Topics   []string
		Messages []json.RawMessage
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || w.Code != 200 || len(result.Topics) != 1 || len(result.Messages) != 1 {
		t.Fatalf("collector history: %d %s %v", w.Code, w.Body.String(), err)
	}
}
