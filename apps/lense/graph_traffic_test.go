package main

import (
	"encoding/json"
	"iot-lense-sense/shared/mqttx/traffic"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestGraphTrafficCacheAndOrigin(t *testing.T) {
	graphTrafficCache.Lock()
	oldItems, oldExpiry := graphTrafficCache.items, graphTrafficCache.expires
	graphTrafficCache.items = map[string]traffic.Snapshot{"fixture": {Available: true, AvgPerMinute: 7}}
	graphTrafficCache.expires = time.Now().Add(time.Minute)
	graphTrafficCache.Unlock()
	defer func() {
		graphTrafficCache.Lock()
		graphTrafficCache.items = oldItems
		graphTrafficCache.expires = oldExpiry
		graphTrafficCache.Unlock()
	}()
	for i := 0; i < 2; i++ {
		r := httptest.NewRequest("GET", "http://localhost:8000/api/graph-traffic", nil)
		r.Header.Set("Origin", "http://localhost:8100")
		w := httptest.NewRecorder()
		apiGraphTraffic(w, r)
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"fixture"`) {
			t.Fatalf("cache lost: %s", w.Body.String())
		}
		if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:8100" {
			t.Fatal("same-host graph access missing")
		}
	}
	r := httptest.NewRequest("GET", "http://localhost:8000/api/graph-traffic", nil)
	r.Header.Set("Origin", "https://untrusted.invalid")
	w := httptest.NewRecorder()
	apiGraphTraffic(w, r)
	if w.Code != 403 {
		t.Fatal("untrusted origin accepted")
	}
}

func TestGraphTrafficCollectsSuccessfulOutputsAndCaches(t *testing.T) {
	var requests atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/api/traffic" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(traffic.Snapshot{Available: true, Count: 50, AvgPerMinute: 10, SampledAt: time.Now(), Last: &traffic.Message{Timestamp: time.Now(), Topic: "target/value", Payload: "42"}})
	}))
	defer upstream.Close()
	loadTopology()
	previous := topologyCache
	defer func() { topologyCache = previous }()
	topologyCache = LenseTopology{Senses: []SenseTopology{{ID: "sense-test", URL: upstream.URL}}, Dispenses: []DispenseTopology{{ID: "dispense-test", URL: upstream.URL, Target: "target-test"}}, Targets: []TargetTopology{{ID: "target-test"}}}
	path := filepath.Join(t.TempDir(), "topology.json")
	raw, _ := json.Marshal(topologyCache)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LENSE_TOPOLOGY_PATH", path)
	graphTrafficCache.Lock()
	oldItems, oldExpiry := graphTrafficCache.items, graphTrafficCache.expires
	graphTrafficCache.items = nil
	graphTrafficCache.Unlock()
	defer func() {
		graphTrafficCache.Lock()
		graphTrafficCache.items = oldItems
		graphTrafficCache.expires = oldExpiry
		graphTrafficCache.Unlock()
	}()
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		apiGraphTraffic(w, httptest.NewRequest("GET", "/api/graph-traffic", nil))
		var body struct {
			Items map[string]traffic.Snapshot `json:"items"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		s := body.Items["target-test"]
		if !s.Available || s.Count != 50 || s.Last.Topic != "target/value" {
			t.Fatalf("target aggregation: %+v", s)
		}
	}
	if requests.Load() != 2 {
		t.Fatalf("repeated graph requests re-polled instances: %d", requests.Load())
	}
}

func TestLiveMessagesIsolationAndUnavailable(t *testing.T) {
	var requests atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/api/messages" {
			t.Errorf("wrong path %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]traffic.Message{{Timestamp: time.Now(), Topic: "selected/value", Payload: "<test>"}})
	}))
	defer upstream.Close()
	loadTopology()
	previous := topologyCache
	defer func() { topologyCache = previous }()
	topologyCache = LenseTopology{Senses: []SenseTopology{{ID: "selected", URL: upstream.URL}, {ID: "other", URL: "http://127.0.0.1:1"}}}
	path := filepath.Join(t.TempDir(), "topology.json")
	raw, _ := json.Marshal(topologyCache)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LENSE_TOPOLOGY_PATH", path)
	for _, tc := range []struct {
		id       string
		code     int
		contains string
	}{
		{"selected", 200, "selected/value"},
		{"http://untrusted.invalid", 404, "Unknown message source"},
		{"other", 200, `"unavailable":["other"]`},
	} {
		w := httptest.NewRecorder()
		apiLiveMessages(w, httptest.NewRequest("GET", "/api/live-messages?id="+tc.id, nil))
		if w.Code != tc.code || !strings.Contains(w.Body.String(), tc.contains) {
			t.Fatalf("%s: %d %s", tc.id, w.Code, w.Body.String())
		}
	}
	if requests.Load() != 1 {
		t.Fatal("polled unrelated source")
	}
}
