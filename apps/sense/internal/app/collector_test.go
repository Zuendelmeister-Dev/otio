package app

import "testing"

func TestPayloadBuilders(t *testing.T) {
	source := SourceConfig{AgentID: "machine-01", Type: "modbus-tcp", Host: "host", Port: 5020, UnitID: 1}
	metric := MetricConfig{Name: "temperature", Register: 0, Scale: 0.1, Unit: "°C", Type: "gauge"}
	payload := metricPayload(source, metric, 235, 23.5)
	if payload["agentId"] != "machine-01" {
		t.Fatalf("unexpected agent id")
	}
	if payload["metric"].(map[string]any)["value"] != 23.5 {
		t.Fatalf("unexpected metric value")
	}
	if statusPayload(source, true, true, "ok")["connected"] != true {
		t.Fatalf("unexpected status payload")
	}
	if errorPayload(source, "boom")["message"] != "boom" {
		t.Fatalf("unexpected error payload")
	}
}

func TestRememberMetric(t *testing.T) {
	state.Lock()
	state.Metrics = map[string]map[string][]MetricPoint{}
	state.Unlock()
	rememberMetric("machine-01", MetricConfig{Name: "temperature", Unit: "°C"}, 23.5)
	state.Lock()
	points := state.Metrics["machine-01"]["temperature"]
	state.Unlock()
	if len(points) != 1 || points[0].Value != 23.5 {
		t.Fatalf("unexpected points: %#v", points)
	}
}
