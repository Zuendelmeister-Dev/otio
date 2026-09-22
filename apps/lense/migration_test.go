package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
)

type migrationDriver struct{}
type migrationConn struct {
	steps      []string
	fail       bool
	committed  bool
	rolledBack bool
}

var currentMigration *migrationConn

func (migrationDriver) Open(string) (driver.Conn, error)     { return currentMigration, nil }
func (c *migrationConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (c *migrationConn) Close() error                        { return nil }
func (c *migrationConn) Begin() (driver.Tx, error)           { c.steps = append(c.steps, "begin"); return c, nil }
func (c *migrationConn) Commit() error                       { c.committed = true; return nil }
func (c *migrationConn) Rollback() error                     { c.rolledBack = true; return nil }
func (c *migrationConn) ExecContext(ctx context.Context, q string, _ []driver.NamedValue) (driver.Result, error) {
	if _, ok := ctx.Deadline(); !ok {
		return nil, errors.New("unbounded migration")
	}
	c.steps = append(c.steps, q)
	if c.fail && strings.HasPrefix(q, "CREATE TABLE") {
		return nil, errors.New("DDL failed")
	}
	return driver.RowsAffected(0), nil
}
func init() { sql.Register("migration-test", migrationDriver{}) }
func TestSchemaMigrationTransaction(t *testing.T) {
	previous := db
	defer func() { db = previous }()
	for _, fail := range []bool{false, true} {
		currentMigration = &migrationConn{fail: fail}
		var err error
		db, err = sql.Open("migration-test", "")
		if err != nil {
			t.Fatal(err)
		}
		err = ensureSchema()
		db.Close()
		c := currentMigration
		if (err != nil) != fail {
			t.Fatalf("unexpected migration result: %v", err)
		}
		if len(c.steps) < 3 || c.steps[0] != "begin" || !strings.Contains(c.steps[1], "pg_advisory_xact_lock") {
			t.Fatal("DDL not protected by transaction lock", c.steps)
		}
		if c.committed == fail || c.rolledBack != fail {
			t.Fatalf("wrong transaction outcome: %+v", c)
		}
	}
}
