package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestProtocolLabExampleTopology(t *testing.T) {
	raw, err := os.ReadFile("../../examples/03-protocol-lab/topology.json")
	if err != nil {
		t.Fatal(err)
	}
	var topology LenseTopology
	if err := json.Unmarshal(raw, &topology); err != nil {
		t.Fatal(err)
	}
	if len(topology.Agents) != 5 || len(topology.Senses) != 1 || len(topology.Presenses) != 1 {
		t.Fatal("incomplete topology")
	}
	configRaw, err := os.ReadFile("../../examples/03-protocol-lab/config/config.json")
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Sources []struct {
			AgentID string `json:"agentId"`
			Type    string `json:"type"`
		} `json:"sources"`
	}
	if err := json.Unmarshal(configRaw, &config); err != nil {
		t.Fatal(err)
	}
	for _, source := range config.Sources {
		found := false
		for _, agent := range topology.Agents {
			if agent.AgentID == source.AgentID && agent.SourceType == source.Type && agent.SenseID == topology.Senses[0].ID && agent.SourceHost != "" {
				found = true
			}
		}
		if !found {
			t.Errorf("source %s has no usable Lense topology entry", source.AgentID)
		}
	}
	if topology.Presenses[0].AgentID == "" {
		t.Fatal("Presense identity missing")
	}
}

func TestProtocolLabProxy(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/read" || r.Method != "POST" {
			t.Errorf("wrong route: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.Copy(w, r.Body)
	}))
	defer upstream.Close()
	w := httptest.NewRecorder()
	protocolLabProxy(upstream.URL).ServeHTTP(w, httptest.NewRequest("POST", "/protocol-lab/api/read", strings.NewReader(`{"value":42}`)))
	if w.Code != 200 || w.Body.String() != `{"value":42}` {
		t.Fatalf("proxy: %d %s", w.Code, w.Body.String())
	}
}

func TestUnavailableProtocolLabExplainsRecovery(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	upstream.Close()
	for _, accept := range []string{"text/html", "application/json"} {
		r := httptest.NewRequest("GET", "/protocol-lab/static/", nil)
		r.Header.Set("Accept", accept)
		w := httptest.NewRecorder()
		protocolLabProxy(upstream.URL).ServeHTTP(w, r)
		if w.Code != 503 || !strings.Contains(w.Body.String(), "Protocol Lab") {
			t.Fatalf("%s: %d %s", accept, w.Code, w.Body.String())
		}
		if accept == "text/html" && !strings.Contains(w.Body.String(), "Back to Lense") {
			t.Fatal("missing recovery navigation")
		}
		if accept == "application/json" && !json.Valid(w.Body.Bytes()) {
			t.Fatal("API error must remain JSON")
		}
	}
}
