package main

import "testing"

func TestComponentStatusCanMarkSenseOffline(t *testing.T) {
	state.Lock()
	state.ComponentStatus = map[string]ComponentStatus{}
	state.Unlock()

	setComponentStatus(ComponentStatus{ID: "iot-sense-opcua-01", Kind: "sense", Connected: false, Healthy: false})
	statuses := componentStatusSnapshot()
	if isComponentHealthy(statuses, "iot-sense-opcua-01", true) {
		t.Fatal("offline Sense component must not be treated as healthy")
	}
}

func TestConfiguredComponentProbeTargetsIncludeSenseAndDispense(t *testing.T) {
	targets := configuredComponentProbeTargets()
	seen := map[string]bool{}
	for _, target := range targets {
		seen[target.ID] = true
		if target.ProbeURL == "" {
			t.Fatalf("probe target %s has empty probe URL", target.ID)
		}
	}
	for _, want := range []string{"iot-sense-modbus-01", "iot-sense-opcua-01", "iot-dispense-modbus-01", "iot-dispense-opcua-01"} {
		if !seen[want] {
			t.Fatalf("expected probe target %s", want)
		}
	}
}
