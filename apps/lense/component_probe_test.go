package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthDistinguishesReachabilityFromWorkingConnections(t *testing.T) {
	for _, tc := range []struct {
		name, kind, body string
		healthy          bool
	}{
		{"forwarder disconnected", "dispense", `{"inputBroker":{"connected":false},"outputBroker":{"connected":false}}`, false},
		{"HTTP output failed", "dispense", `{"inputBroker":{"connected":true},"outputBroker":{"connected":true},"output":{"type":"http","connected":false}}`, false},
		{"HTTP output accepted", "dispense", `{"inputBroker":{"connected":true},"output":{"type":"http","connected":true}}`, true},
		{"legacy forwarder ready", "dispense", `{"inputBroker":{"connected":true},"outputBroker":{"connected":true}}`, true},
		{"invalid collector response", "sense", `not json`, false},
		{"collector disconnected", "sense", `{"broker":{"connected":false}}`, false},
		{"collector ready", "sense", `{"broker":{"connected":true}}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(tc.body)) }))
			defer server.Close()
			connected, healthy, message := probeComponent(*server.Client(), componentProbeTarget{Kind: tc.kind, ProbeURL: server.URL})
			if !connected || healthy != tc.healthy || message == "" {
				t.Fatalf("connected=%v healthy=%v message=%q", connected, healthy, message)
			}
		})
	}
}
