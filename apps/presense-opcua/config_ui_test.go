package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSimulatorConfigValidationPersistenceAndSource(t *testing.T) {
	old := currentSimulatorSettings()
	t.Cleanup(func() { simulatorConfig.Lock(); simulatorConfig.value = old; simulatorConfig.Unlock() })
	path := filepath.Join(t.TempDir(), "simulator.json")
	t.Setenv("PRESENSE_CONFIG_PATH", path)
	t.Setenv("OTIO_CONFIG_WRITE_TOKEN", "test-token")
	initSimulatorSettings(simulatorSettings{Protocol: "modbus-tcp", GeneratorMode: "sine", Temperature: 23})
	mux := http.NewServeMux()
	registerSimulatorConfig(mux, "machine-1", "presense-modbus-01", 5020)
	for _, tc := range []struct {
		body, token string
		status      int
	}{
		{`{"protocol":"modbus-tcp","generatorMode":"constant","temperature":31.2}`, "", 401},
		{`{"protocol":"opcua","generatorMode":"constant","temperature":31.2}`, "test-token", 422},
		{`{"protocol":"modbus-tcp","generatorMode":"invalid"}`, "test-token", 422},
		{`{"protocol":"modbus-tcp","generatorMode":"constant","temperature":-1}`, "test-token", 422},
		{`{"protocol":"modbus-tcp","generatorMode":"constant","unknown":2}`, "test-token", 400},
		{`{} {}`, "test-token", 400},
		{`{"protocol":"modbus-tcp","generatorMode":"constant","temperature":31.2}`, "test-token", 200},
	} {
		r := httptest.NewRequest("PUT", "/api/config", strings.NewReader(tc.body))
		r.Header.Set("X-OTIO-Config-Token", tc.token)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("%s: %d %s", tc.body, w.Code, w.Body.String())
		}
	}
	if s := currentSimulatorSettings(); s.Temperature != 31.2 || s.GeneratorMode != "constant" {
		t.Fatalf("not applied: %+v", s)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var saved simulatorSettings
	if err = json.Unmarshal(data, &saved); err != nil || saved.Temperature != 31.2 {
		t.Fatalf("not saved: %s %v", data, err)
	}
	initSimulatorSettings(simulatorSettings{Protocol: "modbus-tcp", GeneratorMode: "sine"})
	if currentSimulatorSettings().Temperature != 31.2 {
		t.Fatal("persistent settings not restored")
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/config", nil))
	var response struct {
		Source struct {
			Host    string
			Port    int
			Metrics []struct {
				Register int
				Scale    float64
			}
		}
	}
	if err = json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Source.Host != "presense-modbus-01" || response.Source.Port != 5020 || len(response.Source.Metrics) != 6 || response.Source.Metrics[0].Scale != 0.1 {
		t.Fatalf("wrong source: %s", w.Body.String())
	}
}

func TestOPCUASimulatorSource(t *testing.T) {
	source := simulatorSource(simulatorSettings{Protocol: "opcua"}, "ua-1", "presense-opcua-01", 4840)
	raw, _ := json.Marshal(source)
	if !strings.Contains(string(raw), `"nodeId":"ns=2;s=Machine.Temperature"`) {
		t.Fatal(string(raw))
	}
}
