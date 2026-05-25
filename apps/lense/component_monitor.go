package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type ComponentStatus struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Label     string `json:"label"`
	URL       string `json:"url"`
	Connected bool   `json:"connected"`
	Healthy   bool   `json:"healthy"`
	LastSeen  string `json:"lastSeen"`
	Message   string `json:"message"`
}

func setComponentStatus(status ComponentStatus) {
	if status.ID == "" {
		return
	}
	if status.LastSeen == "" {
		status.LastSeen = nowISO()
	}
	state.Lock()
	if state.ComponentStatus == nil {
		state.ComponentStatus = map[string]ComponentStatus{}
	}
	state.ComponentStatus[status.ID] = status
	state.Unlock()
}

func componentStatusSnapshot() map[string]ComponentStatus {
	state.Lock()
	defer state.Unlock()
	result := map[string]ComponentStatus{}
	for key, value := range state.ComponentStatus {
		result[key] = value
	}
	return result
}

func startComponentMonitor() {
	seedConfiguredComponentStatus()
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			pollConfiguredComponents()
			<-ticker.C
		}
	}()
}

func seedConfiguredComponentStatus() {
	for _, item := range configuredComponentProbeTargets() {
		setComponentStatus(ComponentStatus{ID: item.ID, Kind: item.Kind, Label: item.Label, URL: item.URL, Connected: false, Healthy: false, Message: "Waiting for first health check"})
	}
}

func pollConfiguredComponents() {
	client := http.Client{Timeout: 1200 * time.Millisecond}
	for _, item := range configuredComponentProbeTargets() {
		ok, message := probeComponent(client, item.ProbeURL)
		setComponentStatus(ComponentStatus{ID: item.ID, Kind: item.Kind, Label: item.Label, URL: item.URL, Connected: ok, Healthy: ok, Message: message})
	}
}

type componentProbeTarget struct {
	ID       string
	Kind     string
	Label    string
	URL      string
	ProbeURL string
}

func configuredComponentProbeTargets() []componentProbeTarget {
	raw := loadRawTopology()
	var result []componentProbeTarget
	addItems := func(group string, kind string) {
		items, _ := raw[group].([]any)
		for _, value := range items {
			item, _ := value.(map[string]any)
			if item == nil {
				continue
			}
			id := stringValue(item, "id")
			if id == "" {
				id = stringValue(item, "agentId")
			}
			if id == "" {
				continue
			}
			label := stringValue(item, "label")
			if label == "" {
				label = id
			}
			url := stringValue(item, "url")
			probeBase := stringValue(item, "internalUrl")
			if probeBase == "" {
				probeBase = url
			}
			probePath := "/api/status"
			if kind == "presense" && strings.Contains(strings.ToLower(stringValue(item, "sourceType")), "opcua") {
				probePath = "/health"
			}
			if probeBase == "" || strings.Contains(probeBase, "127.0.0.1") && kind == "presense" {
				// Presense entries in the demo topology need an internalUrl for container-to-container probing.
				// If no internal URL is present, keep the seeded status instead of probing the Lense container itself.
				continue
			}
			result = append(result, componentProbeTarget{ID: id, Kind: kind, Label: label, URL: url, ProbeURL: strings.TrimRight(probeBase, "/") + probePath})
		}
	}
	addItems("senses", "sense")
	addItems("dispenses", "dispense")
	addItems("presenses", "presense")
	return result
}

func stringValue(item map[string]any, key string) string {
	if value, ok := item[key].(string); ok {
		return value
	}
	return ""
}

func probeComponent(client http.Client, probeURL string) (bool, string) {
	response, err := client.Get(probeURL)
	if err != nil {
		return false, err.Error()
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return false, response.Status
	}
	return true, "Health endpoint reachable"
}

func storeSenseComponentStatus(raw string) {
	var payload struct {
		Timestamp       string         `json:"timestamp"`
		SenseID         string         `json:"senseId"`
		BrokerConnected bool           `json:"brokerConnected"`
		Sources         map[string]any `json:"sources"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return
	}
	if payload.SenseID == "" {
		return
	}
	message := "Sense heartbeat received"
	if !payload.BrokerConnected {
		message = "Sense heartbeat received, but broker is reported disconnected"
	}
	setComponentStatus(ComponentStatus{ID: payload.SenseID, Kind: "sense", Label: payload.SenseID, Connected: true, Healthy: payload.BrokerConnected, LastSeen: payload.Timestamp, Message: message})
}

func isComponentHealthy(statuses map[string]ComponentStatus, id string, defaultValue bool) bool {
	if status, ok := statuses[id]; ok {
		return status.Connected && status.Healthy
	}
	return defaultValue
}
