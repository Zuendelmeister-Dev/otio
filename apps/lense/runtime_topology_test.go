package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestAPIRuntimeReturnsFullComponentConfiguration(t *testing.T) {
	tmp := t.TempDir()
	topologyPath := filepath.Join(tmp, "topology.json")
	topologyJSON := `{
		"senses": [{
			"id": "iot-sense-modbus-01",
			"label": "IoT Sense Modbus",
			"url": "http://127.0.0.1:8100",
			"sourceType": "modbus-tcp",
			"configuration": {
				"broker": {"clientId": "iot-sense-modbus-01", "host": "mqtt", "port": 1883, "topicPrefix": "iot-lense"},
				"pollIntervalMs": 1000,
				"healthTimeoutSeconds": 300,
				"sources": [{
					"agentId": "modbus-machine-01",
					"host": "presense-modbus-01",
					"port": 5020,
					"type": "modbus-tcp",
					"unitId": 1,
					"metrics": [{"name": "temperature", "register": 0, "scale": 0.1, "type": "gauge", "unit": "°C"}]
				}]
			}
		}],
		"presenses": [{
			"agentId": "modbus-machine-01",
			"sourceType": "modbus-tcp",
			"sourceHost": "presense-modbus-01",
			"senseId": "iot-sense-modbus-01",
			"url": "http://127.0.0.1:8301",
			"configuration": {
				"deviceId": "modbus-machine-01",
				"protocol": "modbus-tcp",
				"registers": [{"name": "temperature", "register": 0, "scale": 0.1}]
			}
		}]
	}`
	if err := os.WriteFile(topologyPath, []byte(topologyJSON), 0o600); err != nil {
		t.Fatalf("write topology: %v", err)
	}

	t.Setenv("LENSE_TOPOLOGY_PATH", topologyPath)

	request := httptest.NewRequest("GET", "/api/runtime", nil)
	response := httptest.NewRecorder()
	apiRuntime(response, request)

	if response.Code != 200 {
		t.Fatalf("unexpected status: %d", response.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	topology, ok := payload["topology"].(map[string]any)
	if !ok {
		t.Fatalf("topology missing or wrong type: %#v", payload["topology"])
	}

	senses := topology["senses"].([]any)
	sense := senses[0].(map[string]any)
	config := sense["configuration"].(map[string]any)

	if _, ok := config["broker"].(map[string]any); !ok {
		t.Fatalf("broker configuration missing: %#v", config)
	}

	sources := config["sources"].([]any)
	source := sources[0].(map[string]any)
	metrics := source["metrics"].([]any)
	metric := metrics[0].(map[string]any)

	if metric["register"].(float64) != 0 {
		t.Fatalf("expected full metric register to survive, got %#v", metric)
	}

	presenses := topology["presenses"].([]any)
	presense := presenses[0].(map[string]any)
	presenseConfig := presense["configuration"].(map[string]any)
	registers := presenseConfig["registers"].([]any)

	if len(registers) != 1 {
		t.Fatalf("expected full presense registers to survive, got %#v", presenseConfig)
	}
}
