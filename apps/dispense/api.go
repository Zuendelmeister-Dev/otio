package main

import (
	"encoding/json"
	"net/http"
)

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func apiStatus(w http.ResponseWriter, r *http.Request) {
	state.Lock()
	defer state.Unlock()
	writeJSON(w, map[string]any{
		"instanceId":   state.Config.InstanceID,
		"sourceFilter": state.Config.SourceFilter,
		"inputBroker": map[string]any{
			"host":      state.Config.InputBroker.Host,
			"port":      state.Config.InputBroker.Port,
			"connected": state.InputConnected,
			"lastInput": state.LastInput,
		},
		"outputBroker": map[string]any{
			"host":       state.Config.OutputBroker.Host,
			"port":       state.Config.OutputBroker.Port,
			"connected":  state.OutputConnected,
			"lastOutput": state.LastOutput,
		},
		"stats": map[string]any{
			"received":       state.ReceivedCount,
			"forwarded":      state.ForwardedCount,
			"dropped":        state.DroppedCount,
			"topicsObserved": len(state.Topics),
		},
	})
}

func apiMetrics(w http.ResponseWriter, r *http.Request) {
	state.Lock()
	defer state.Unlock()
	writeJSON(w, state.Metrics)
}

func apiLogs(w http.ResponseWriter, r *http.Request) {
	state.Lock()
	defer state.Unlock()
	writeJSON(w, map[string]any{"items": state.Logs})
}

func apiRuntime(w http.ResponseWriter, r *http.Request) {
	state.Lock()
	defer state.Unlock()
	writeJSON(w, map[string]any{"config": state.Config})
}
