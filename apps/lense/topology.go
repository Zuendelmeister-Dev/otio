package main

import (
	"encoding/json"
	"os"
	"sync"
)

type SenseTopology struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	URL        string `json:"url"`
	SourceType string `json:"sourceType"`
}

type AgentTopology struct {
	AgentID    string `json:"agentId"`
	SourceType string `json:"sourceType"`
	SourceHost string `json:"sourceHost"`
	SenseID    string `json:"senseId"`
	URL        string `json:"url"`
}

type DispenseTopology struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	URL        string `json:"url"`
	SourceType string `json:"sourceType"`
	Target     string `json:"target"`
}

type TargetTopology struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	ExternalPort int    `json:"externalPort"`
}

type LenseTopology struct {
	Senses    []SenseTopology    `json:"senses"`
	Agents    []AgentTopology    `json:"agents"`
	Presenses []AgentTopology    `json:"presenses"`
	Dispenses []DispenseTopology `json:"dispenses"`
	Targets   []TargetTopology   `json:"targets"`
}

var topologyOnce sync.Once
var topologyCache LenseTopology

func loadTopology() LenseTopology {
	topologyOnce.Do(func() {
		path := getenv("LENSE_TOPOLOGY_PATH", "/app/config/topology.json")
		raw, err := os.ReadFile(path)
		if err == nil {
			_ = json.Unmarshal(raw, &topologyCache)
		}
		if len(topologyCache.Agents) == 0 {
			topologyCache = LenseTopology{
				Senses: []SenseTopology{
					{ID: "iot-sense-modbus-01", Label: "IoT Sense Modbus", URL: "http://127.0.0.1:8100", SourceType: "modbus-tcp"},
					{ID: "iot-sense-opcua-01", Label: "IoT Sense OPC UA", URL: "http://127.0.0.1:8101", SourceType: "opcua"},
				},
				Dispenses: []DispenseTopology{
					{ID: "iot-dispense-modbus-01", Label: "IoT Dispense Modbus", URL: "http://127.0.0.1:8200", SourceType: "modbus-tcp", Target: "dispense-target-mqtt"},
					{ID: "iot-dispense-opcua-01", Label: "IoT Dispense OPC UA", URL: "http://127.0.0.1:8201", SourceType: "opcua", Target: "dispense-target-mqtt"},
				},
				Targets: []TargetTopology{
					{ID: "dispense-target-mqtt", Label: "Dispense Target MQTT", Host: "dispense-broker", Port: 1883, ExternalPort: 1884},
				},
				Agents: []AgentTopology{
					{AgentID: "modbus-machine-01", SourceType: "modbus-tcp", SourceHost: "presense-modbus-01", SenseID: "iot-sense-modbus-01"},
					{AgentID: "modbus-machine-02", SourceType: "modbus-tcp", SourceHost: "presense-modbus-02", SenseID: "iot-sense-modbus-01"},
					{AgentID: "modbus-machine-03", SourceType: "modbus-tcp", SourceHost: "presense-modbus-03", SenseID: "iot-sense-modbus-01"},
					{AgentID: "opcua-machine-01", SourceType: "opcua", SourceHost: "presense-opcua-01", SenseID: "iot-sense-opcua-01"},
					{AgentID: "opcua-machine-02", SourceType: "opcua", SourceHost: "presense-opcua-02", SenseID: "iot-sense-opcua-01"},
				},
			}
		}
	})
	return topologyCache
}

func configuredAgentIDs() []string {
	topology := loadTopology()
	ids := make([]string, 0, len(topology.Agents))
	for _, agent := range topology.Agents {
		ids = append(ids, agent.AgentID)
	}
	return ids
}

func configuredAgentSet() map[string]AgentTopology {
	topology := loadTopology()
	result := map[string]AgentTopology{}
	for _, agent := range topology.Agents {
		result[agent.AgentID] = agent
	}
	return result
}

func configuredSenseSet() map[string]SenseTopology {
	topology := loadTopology()
	result := map[string]SenseTopology{}
	for _, sense := range topology.Senses {
		result[sense.ID] = sense
	}
	return result
}

func isConfiguredAgent(agentID string) bool {
	_, ok := configuredAgentSet()[agentID]
	return ok
}

func sqlInPlaceholders(count int, start int) string {
	result := ""
	for i := 0; i < count; i++ {
		if i > 0 {
			result += ","
		}
		result += "$" + itoa(start+i)
	}
	return result
}

func itoa(value int) string {
	digits := "0123456789"
	if value == 0 {
		return "0"
	}
	buf := []byte{}
	for value > 0 {
		buf = append([]byte{digits[value%10]}, buf...)
		value /= 10
	}
	return string(buf)
}

func stringArgs(values []string) []any {
	args := make([]any, len(values))
	for i, value := range values {
		args[i] = value
	}
	return args
}

// loadRawTopology returns the exact topology JSON used by the UI.
// This intentionally keeps nested configuration objects untouched.
func loadRawTopology() map[string]any {
	path := getenv("LENSE_TOPOLOGY_PATH", "/app/config/topology.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return topologyToRawMap(loadTopology())
	}

	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return topologyToRawMap(loadTopology())
	}
	return result
}

// topologyToRawMap is the fallback path used when no topology file is present.
func topologyToRawMap(topology LenseTopology) map[string]any {
	raw, err := json.Marshal(topology)
	if err != nil {
		return map[string]any{}
	}

	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return map[string]any{}
	}
	return result
}
