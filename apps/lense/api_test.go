package main

import (
	"database/sql"
	"testing"
	"time"
)

func TestCalcStats(t *testing.T) {
	avg, min, max := calcStats([]float64{2, 4, 6})
	if avg != 4 || min != 2 || max != 6 {
		t.Fatalf("calcStats = %v %v %v", avg, min, max)
	}
	avg, min, max = calcStats(nil)
	if avg != 0 || min != 0 || max != 0 {
		t.Fatalf("calcStats empty = %v %v %v", avg, min, max)
	}
}

func TestNormalizeAggregation(t *testing.T) {
	for _, input := range []string{"avg", "min", "max", "sum"} {
		if got := normalizeAggregation(input); got != input {
			t.Fatalf("normalizeAggregation(%q) = %q", input, got)
		}
	}
	if got := normalizeAggregation("drop table"); got != "avg" {
		t.Fatalf("unexpected fallback aggregation: %q", got)
	}
}

func TestBuildTree(t *testing.T) {
	tree := buildTree([]string{"iot-lense/machine-01/metrics/temperature", "iot-lense/machine-01/status"})
	root, ok := tree["iot-lense"].(map[string]any)
	if !ok {
		t.Fatalf("missing iot-lense root: %#v", tree)
	}
	machine, ok := root["machine-01"].(map[string]any)
	if !ok {
		t.Fatalf("missing machine node: %#v", root)
	}
	if _, ok := machine["metrics"]; !ok {
		t.Fatalf("missing metrics node: %#v", machine)
	}
	if _, ok := machine["status"]; !ok {
		t.Fatalf("missing status node: %#v", machine)
	}
}

func TestSplitTopic(t *testing.T) {
	parts := splitTopic("iot-lense/machine-01/metrics/temperature")
	if len(parts) != 4 || parts[1] != "machine-01" || parts[3] != "temperature" {
		t.Fatalf("unexpected parts: %#v", parts)
	}
}

func TestScanTimeString(t *testing.T) {
	stamp := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	if got := scanTimeString(stamp); got != "2026-05-20T12:00:00Z" {
		t.Fatalf("scanTimeString = %q", got)
	}
}

func TestNullStringValue(t *testing.T) {
	if got := nullStringValue(sql.NullString{String: "x", Valid: true}); got != "x" {
		t.Fatalf("nullStringValue valid = %q", got)
	}
	if got := nullStringValue(sql.NullString{}); got != "" {
		t.Fatalf("nullStringValue invalid = %q", got)
	}
}
