package app

import (
	"os"
	"testing"
)

func TestProtocolLabExample(t *testing.T) {
	raw, err := os.ReadFile("../../../../examples/03-protocol-lab/config/config.json")
	if err != nil {
		t.Fatal(err)
	}
	config, issues := parseAndValidate(string(raw))
	if len(issues) != 0 {
		t.Fatalf("invalid example: %v", issues)
	}
	if len(config.Sources) != 5 {
		t.Fatalf("expected five sources, got %d", len(config.Sources))
	}
	for _, source := range config.Sources {
		for _, metric := range source.Metrics {
			if metric.NodeID != "" && metricAddress(source, metric) != metric.NodeID {
				t.Fatalf("%s: telemetry lost node address", source.AgentID)
			}
		}
	}
}
