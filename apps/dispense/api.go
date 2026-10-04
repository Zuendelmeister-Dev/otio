package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"iot-lense-sense/shared/mqttx"
)

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func apiStatus(w http.ResponseWriter, r *http.Request) {
	state.Lock()
	defer state.Unlock()
	outputHost, outputPort := state.Config.OutputBroker.Host, state.Config.OutputBroker.Port
	outputConnected := mqttx.IsReady(state.OutputClient)
	outputType := state.Config.Output.Type
	if outputType == "" {
		outputType = "mqtt"
	}
	if outputType == "http" {
		if endpoint, err := url.Parse(state.Config.Output.URL); err == nil {
			outputHost = endpoint.Hostname()
			outputPort, _ = strconv.Atoi(endpoint.Port())
			if outputPort == 0 {
				outputPort = 80
				if endpoint.Scheme == "https" {
					outputPort = 443
				}
			}
		}
		outputConnected = state.Output != nil && state.Output.writer.Ready()
	}
	output := map[string]any{
		"type": outputType, "url": publicOutputURL(state.Config.Output.URL),
		"host": outputHost, "port": outputPort, "connected": outputConnected,
		"lastOutput": state.LastOutput,
	}
	writeJSON(w, map[string]any{
		"instanceId":   state.Config.InstanceID,
		"sourceFilter": state.Config.SourceFilter,
		"inputBroker": map[string]any{
			"host":      state.Config.InputBroker.Host,
			"port":      state.Config.InputBroker.Port,
			"connected": mqttx.IsReady(state.InputClient),
			"lastInput": state.LastInput,
		},
		"output":       output,
		"outputBroker": output,
		"stats": map[string]any{
			"received":       state.ReceivedCount,
			"forwarded":      state.ForwardedCount,
			"dropped":        state.DroppedCount,
			"filtered":       state.FilteredCount,
			"pending":        state.PendingCount,
			"retries":        state.RetryCount,
			"uncertain":      state.UncertainCount,
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
	config := state.Config
	config.Output.URL = publicOutputURL(config.Output.URL)
	writeJSON(w, map[string]any{"config": config})
}
