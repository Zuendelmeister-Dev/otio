package main

import (
	"encoding/json"
	"testing"
)

func TestMetricMessageDecode(t *testing.T) {
	raw := []byte(`{"schemaVersion":"1.0","timestamp":"2026-05-20T12:00:00Z","agentId":"machine-01","source":{"type":"modbus-tcp","host":"host","address":"holding-register:0"},"metric":{"name":"temperature","value":23.5,"unit":"°C","type":"gauge"},"quality":{"status":"good"}}`)
	var msg MetricMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatal(err)
	}
	if msg.AgentID != "machine-01" || msg.Metric.Name != "temperature" || msg.Metric.Value != 23.5 {
		t.Fatalf("unexpected message: %#v", msg)
	}
}
