package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"iot-lense-sense/shared/mqttx/traffic"
)

// Only configured instances may be queried; clients cannot supply an upstream URL.
func apiLiveMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(405)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	id := r.URL.Query().Get("id")
	items := []traffic.Message{}
	label, scope := id, "Successful MQTT publications from this instance"
	unavailable := []string{}
	if id == "mqtt" {
		label = "MQTT Broker"
		scope = "Messages observed by this Lense replica's configured MQTT subscription"
		items = messageTraffic.Recent()
	} else {
		ids := map[string]bool{id: true}
		found := false
		for _, target := range loadTopology().Targets {
			if target.ID == id {
				found = true
				label = target.Label
				scope = "Successful Dispense publications to this target"
				ids = map[string]bool{}
				for _, d := range loadTopology().Dispenses {
					if d.Target == id {
						ids[d.ID] = true
					}
				}
				if len(ids) == 0 {
					unavailable = append(unavailable, "No configured publishers")
				}
			}
		}
		for _, target := range configuredComponentProbeTargets() {
			if !ids[target.ID] || (target.Kind != "sense" && target.Kind != "dispense") {
				continue
			}
			found = true
			if target.ID == id {
				label = target.Label
			}
			endpoint := strings.TrimSuffix(target.ProbeURL, "/api/status") + "/api/messages"
			req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
			var messages []traffic.Message
			if err == nil {
				client := http.Client{Timeout: 3 * time.Second}
				response, e := client.Do(req)
				err = e
				if err == nil {
					if response.StatusCode == 200 {
						err = json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&messages)
					} else {
						err = fmt.Errorf("message source returned HTTP %d", response.StatusCode)
					}
					response.Body.Close()
				}
			}
			if err != nil {
				unavailable = append(unavailable, target.ID)
				continue
			}
			if len(messages) > 50 {
				messages = messages[:50]
			}
			items = append(items, messages...)
		}
		if !found {
			http.Error(w, "Unknown message source", 404)
			return
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Timestamp.After(items[j].Timestamp) })
	if len(items) > 50 {
		items = items[:50]
	}
	writeJSON(w, map[string]any{"items": items, "label": label, "scope": scope, "unavailable": unavailable, "sampledAt": time.Now().UTC()})
}
