package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/csv"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type exportDriver struct{}
type exportConn struct{ fail bool }
type exportRows struct{ read bool }

func (exportDriver) Open(mode string) (driver.Conn, error) { return exportConn{mode == "fail"}, nil }
func (exportConn) Prepare(string) (driver.Stmt, error)     { return nil, driver.ErrSkip }
func (exportConn) Close() error                            { return nil }
func (exportConn) Begin() (driver.Tx, error)               { return nil, driver.ErrSkip }
func (c exportConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	if c.fail {
		return nil, errors.New("database offline")
	}
	return &exportRows{}, nil
}
func (*exportRows) Columns() []string {
	return []string{"ts", "agent_id", "metric_name", "metric_value", "metric_text", "metric_unit"}
}
func (*exportRows) Close() error { return nil }
func (r *exportRows) Next(values []driver.Value) error {
	if r.read {
		return io.EOF
	}
	r.read = true
	copy(values, []driver.Value{time.Unix(0, 0).UTC(), "device,1", "state", nil, "line 1\n\"line 2\"", "=unsafe()"})
	return nil
}

func TestExportEscapesTextAndReportsDatabaseFailure(t *testing.T) {
	sql.Register("export-fixture", exportDriver{})
	original := db
	t.Cleanup(func() { db = original })
	for _, mode := range []string{"good", "fail"} {
		var err error
		db, err = sql.Open("export-fixture", mode)
		if err != nil {
			t.Fatal(err)
		}
		recorder := httptest.NewRecorder()
		apiExport(recorder, httptest.NewRequest("GET", "/api/export?format=csv", nil))
		db.Close()
		if mode == "fail" {
			if recorder.Code != 503 {
				t.Fatalf("failure returned %d", recorder.Code)
			}
			continue
		}
		if recorder.Code != 200 {
			t.Fatalf("export returned %d", recorder.Code)
		}
		rows, err := csv.NewReader(strings.NewReader(recorder.Body.String())).ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 2 || rows[1][1] != "device,1" || rows[1][4] != "line 1\n\"line 2\"" || rows[1][5] != "'=unsafe()" {
			t.Fatalf("CSV roundtrip: %#v", rows)
		}
	}
}
