package app

import "testing"

func TestMetricPayloadAcceptsStringValue(t *testing.T) {
	source := SourceConfig{AgentID: "opcua-machine-01", Type: "opcua", Host: "presense-opcua-01", Port: 4840}
	metric := MetricConfig{Name: "state", NodeID: "ns=2;s=Machine.State", Type: "string"}

	payload := metricPayload(source, metric, map[string]any{"nodeId": metric.NodeID, "value": "running"}, "running")
	metricPayload, ok := payload["metric"].(map[string]any)
	if !ok {
		t.Fatalf("metric payload missing: %#v", payload)
	}
	if metricPayload["value"] != "running" {
		t.Fatalf("metric value = %#v, want running", metricPayload["value"])
	}
	if metricPayload["type"] != "string" {
		t.Fatalf("metric type = %#v, want string", metricPayload["type"])
	}
}
