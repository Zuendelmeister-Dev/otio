package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// A deliberately small SELECT grammar. Rebuild SQL from tokens, never execute
// arbitrary user SQL (including function calls, comments or multiple statements).
var viewerToken = regexp.MustCompile(`^(?:'(?:[^']|'')*'|"(?:[^"]|"")*"|[A-Za-z_][A-Za-z_0-9]*|-?[0-9]+(?:\.[0-9]+)?|<=|>=|<>|!=|[=<>*,.;])`)
var viewerIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z_0-9]*$`)

type viewerParser struct {
	tokens []string
	pos    int
	args   []any
}

func (p *viewerParser) take(s string) bool {
	if p.pos < len(p.tokens) && strings.EqualFold(p.tokens[p.pos], s) {
		p.pos++
		return true
	}
	return false
}
func (p *viewerParser) identifier() (string, error) {
	if p.pos == len(p.tokens) {
		return "", fmt.Errorf("expected an identifier")
	}
	s := p.tokens[p.pos]
	p.pos++
	if strings.HasPrefix(s, `"`) {
		return s, nil
	}
	if !viewerIdentifier.MatchString(s) {
		return "", fmt.Errorf("expected a column or table name")
	}
	return `"` + strings.ToLower(s) + `"`, nil
}
func parseViewerQuery(input string) (string, []any, int, error) {
	fail := func() (string, []any, int, error) {
		return "", nil, 0, fmt.Errorf("Use SELECT columns FROM public.table [WHERE column operator value AND ...] [ORDER BY column ASC|DESC] [LIMIT 1..500]. Functions, joins, comments and multiple statements are not supported.")
	}
	p := viewerParser{}
	rest := strings.TrimSpace(input)
	for rest != "" {
		t := viewerToken.FindString(rest)
		if t == "" {
			return fail()
		}
		p.tokens = append(p.tokens, t)
		rest = strings.TrimSpace(rest[len(t):])
	}
	if !p.take("SELECT") {
		return fail()
	}
	columns := []string{}
	if p.take("*") {
		columns = append(columns, "*")
	} else {
		for {
			c, e := p.identifier()
			if e != nil {
				return fail()
			}
			columns = append(columns, c)
			if !p.take(",") {
				break
			}
		}
	}
	if !p.take("FROM") {
		return fail()
	}
	table, e := p.identifier()
	if e != nil {
		return fail()
	}
	if p.take(".") {
		if table != `"public"` {
			return fail()
		}
		table, e = p.identifier()
		if e != nil {
			return fail()
		}
	}
	query := "SELECT " + strings.Join(columns, ",") + " FROM public." + table
	if p.take("WHERE") {
		conditions := []string{}
		for {
			col, e := p.identifier()
			if e != nil {
				return fail()
			}
			if p.take("IS") {
				op := " IS "
				if p.take("NOT") {
					op += "NOT "
				}
				if !p.take("NULL") {
					return fail()
				}
				conditions = append(conditions, col+op+"NULL")
			} else {
				if p.pos == len(p.tokens) {
					return fail()
				}
				op := strings.ToUpper(p.tokens[p.pos])
				p.pos++
				switch op {
				case "=", "!=", "<>", "<", ">", "<=", ">=", "LIKE", "ILIKE":
				default:
					return fail()
				}
				if p.pos == len(p.tokens) {
					return fail()
				}
				value := p.tokens[p.pos]
				p.pos++
				var arg any
				if strings.HasPrefix(value, "'") {
					arg = strings.ReplaceAll(value[1:len(value)-1], "''", "'")
				} else if strings.EqualFold(value, "true") || strings.EqualFold(value, "false") {
					arg = strings.EqualFold(value, "true")
				} else {
					if _, e := strconv.ParseFloat(value, 64); e != nil {
						return fail()
					}
					arg = value
				}
				p.args = append(p.args, arg)
				conditions = append(conditions, fmt.Sprintf("%s %s $%d", col, op, len(p.args)))
			}
			if !p.take("AND") {
				break
			}
		}
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	if p.take("ORDER") {
		if !p.take("BY") {
			return fail()
		}
		order := []string{}
		for {
			c, e := p.identifier()
			if e != nil {
				return fail()
			}
			if p.take("DESC") {
				c += " DESC"
			} else {
				p.take("ASC")
				c += " ASC"
			}
			order = append(order, c)
			if !p.take(",") {
				break
			}
		}
		query += " ORDER BY " + strings.Join(order, ",")
	}
	limit := 200
	if p.take("LIMIT") {
		if p.pos == len(p.tokens) {
			return fail()
		}
		limit, e = strconv.Atoi(p.tokens[p.pos])
		p.pos++
		if e != nil || limit < 1 || limit > 500 {
			return fail()
		}
	}
	p.take(";")
	if p.pos != len(p.tokens) {
		return fail()
	}
	// One extra row tells the client whether the displayed/exported result is capped.
	return fmt.Sprintf("%s LIMIT %d", query, limit+1), p.args, limit, nil
}

func viewerError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
func apiDatabaseTables(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.Header().Set("Allow", "GET")
		viewerError(w, 405, fmt.Errorf("GET required"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	rows, err := db.QueryContext(ctx, `SELECT c.table_name,c.column_name,c.data_type FROM information_schema.columns c JOIN information_schema.tables t ON t.table_schema=c.table_schema AND t.table_name=c.table_name WHERE c.table_schema='public' AND t.table_type='BASE TABLE' ORDER BY c.table_name,c.ordinal_position`)
	if err != nil {
		viewerError(w, 503, err)
		return
	}
	defer rows.Close()
	type column struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	tables := map[string][]column{}
	for rows.Next() {
		var table string
		var c column
		if err = rows.Scan(&table, &c.Name, &c.Type); err != nil {
			viewerError(w, 500, err)
			return
		}
		tables[table] = append(tables[table], c)
	}
	if err = rows.Err(); err != nil {
		viewerError(w, 500, err)
		return
	}
	writeJSON(w, map[string]any{"database": getenv("POSTGRES_DB", "iotdb"), "tables": tables})
}
func apiDatabaseQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		viewerError(w, 405, fmt.Errorf("POST required"))
		return
	}
	var request struct {
		SQL string `json:"sql"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		viewerError(w, 400, err)
		return
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		viewerError(w, 400, fmt.Errorf("expected one JSON object"))
		return
	}
	query, args, limit, err := parseViewerQuery(request.SQL)
	if err != nil {
		viewerError(w, 400, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		viewerError(w, 503, err)
		return
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SET LOCAL statement_timeout = '4s'`); err != nil {
		viewerError(w, 500, err)
		return
	}
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		viewerError(w, 400, err)
		return
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		viewerError(w, 500, err)
		return
	}
	result := [][]any{}
	truncated := false
	size := 0
	for rows.Next() {
		if len(result) == limit {
			truncated = true
			break
		}
		values := make([]any, len(columns))
		dest := make([]any, len(columns))
		for i := range values {
			dest[i] = &values[i]
		}
		if err = rows.Scan(dest...); err != nil {
			viewerError(w, 500, err)
			return
		}
		for i, v := range values {
			if b, ok := v.([]byte); ok {
				values[i] = string(b)
			}
		}
		encoded, _ := json.Marshal(values)
		size += len(encoded)
		if size > 2*1024*1024 {
			truncated = true
			break
		}
		result = append(result, values)
	}
	if err = rows.Err(); err != nil {
		viewerError(w, 400, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, map[string]any{"columns": columns, "rows": result, "truncated": truncated, "limit": limit})
}
