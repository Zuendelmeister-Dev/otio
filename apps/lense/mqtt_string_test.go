package main

import (
	"encoding/json"
	"testing"
)

func TestMetricMessageAcceptsStringValue(t *testing.T) {
	raw := []byte(`{
		"schemaVersion": "1.0",
		"timestamp": "2026-01-01T00:00:00Z",
		"agentId": "opcua-machine-01",
		"source": {"type": "opcua", "host": "presense-opcua-01", "address": "ns=2;s=Machine.State"},
		"metric": {"name": "state", "value": "running", "unit": "", "type": "string"},
		"quality": {"status": "good"}
	}`)

	var message MetricMessage
	if err := json.Unmarshal(raw, &message); err != nil {
		t.Fatalf("unmarshal metric message: %v", err)
	}
	if message.Metric.Name != "state" {
		t.Fatalf("metric name = %s, want state", message.Metric.Name)
	}
	if message.Metric.Value != "running" {
		t.Fatalf("metric value = %#v, want running", message.Metric.Value)
	}
}
