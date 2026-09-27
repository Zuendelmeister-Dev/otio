package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"iot-lense-sense/shared/mqttx/traffic"
)

var graphTrafficCache struct {
	sync.Mutex
	expires time.Time
	items   map[string]traffic.Snapshot
}

// All graph views share this cache: opening more tabs does not multiply upstream polling.
func apiGraphTraffic(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(405)
		return
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, e := url.Parse(origin)
		base, _ := url.Parse("http://" + r.Host)
		allowed := e == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() == base.Hostname()
		if !allowed && e == nil {
			for _, target := range configuredComponentProbeTargets() {
				configured, err := url.Parse(target.URL)
				if err == nil && configured.Scheme == u.Scheme && configured.Host == u.Host {
					allowed = true
					break
				}
			}
		}
		if !allowed {
			http.Error(w, "Graph traffic is available to same-host instances", 403)
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
	}
	w.Header().Set("Cache-Control", "no-store")
	graphTrafficCache.Lock()
	defer graphTrafficCache.Unlock()
	if graphTrafficCache.items == nil || time.Now().After(graphTrafficCache.expires) {
		graphTrafficCache.expires = time.Now().Add(time.Minute)
		graphTrafficCache.items = collectGraphTraffic()
	}
	writeJSON(w, map[string]any{"items": graphTrafficCache.items, "refreshSeconds": 60})
}

func collectGraphTraffic() map[string]traffic.Snapshot {
	items := map[string]traffic.Snapshot{}
	broker := messageTraffic.Snapshot()
	broker.Scope = "MQTT messages observed by Lense (configured subscription, including status)"
	items["mqtt"] = broker
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, target := range configuredComponentProbeTargets() {
		if target.Kind != "sense" && target.Kind != "dispense" {
			continue
		}
		wg.Add(1)
		go func(target componentProbeTarget) {
			defer wg.Done()
			snapshot := traffic.Snapshot{Available: false, SampledAt: time.Now().UTC(), Scope: "Successful publications; instance unavailable"}
			endpoint := strings.TrimSuffix(target.ProbeURL, "/api/status") + "/api/traffic"
			client := http.Client{Timeout: 3 * time.Second}
			response, err := client.Get(endpoint)
			if err == nil {
				defer response.Body.Close()
				if response.StatusCode == 200 {
					var candidate traffic.Snapshot
					if json.NewDecoder(io.LimitReader(response.Body, 16384)).Decode(&candidate) == nil && !candidate.SampledAt.IsZero() {
						snapshot = candidate
						snapshot.Scope = "Successful MQTT publications from this " + target.Kind + " instance"
					}
				}
			}
			mu.Lock()
			items[target.ID] = snapshot
			mu.Unlock()
		}(target)
	}
	wg.Wait()
	topology := loadTopology()
	for _, target := range topology.Targets {
		sum := traffic.Snapshot{Available: true, SampledAt: time.Now().UTC(), Scope: "Successful Dispense publications to this target (observed traffic)"}
		found := false
		for _, d := range topology.Dispenses {
			if d.Target != target.ID {
				continue
			}
			found = true
			s, ok := items[d.ID]
			if !ok || !s.Available {
				sum.Available = false
				continue
			}
			sum.Count += s.Count
			sum.AvgPerMinute += s.AvgPerMinute
			if sum.Since.IsZero() || s.Since.Before(sum.Since) {
				sum.Since = s.Since
			}
			if s.Last != nil && (sum.Last == nil || s.Last.Timestamp.After(sum.Last.Timestamp)) {
				sum.Last = s.Last
			}
		}
		sum.Available = sum.Available && found
		items[target.ID] = sum
	}
	return items
}
