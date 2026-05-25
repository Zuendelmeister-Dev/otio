package protocols

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func endpointFromServer(t *testing.T, server *httptest.Server) SourceEndpoint {
	t.Helper()
	host, portString, err := net.SplitHostPort(strings.TrimPrefix(server.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portString)
	if err != nil {
		t.Fatal(err)
	}
	return SourceEndpoint{AgentID: "opcua-machine-01", Type: "opcua", Host: host, Port: port}
}

func TestOPCUAHTTPReaderReadsNodeValue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/read" {
			t.Fatalf("path = %s, want /read", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"nodeId":  r.URL.Query().Get("nodeId"),
			"value":   235.0,
			"quality": "good",
		})
	}))
	defer server.Close()

	reader := OPCUAHTTPReader{}
	value, err := reader.Read(context.Background(), endpointFromServer(t, server), MetricAddress{Name: "temperature", NodeID: "ns=2;s=temperature", Scale: 0.1})
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if value.Name != "temperature" || value.Value != 23.5 {
		t.Fatalf("unexpected value: %#v", value)
	}
}

func TestOPCUAHTTPReaderSubscribesToNodeValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/subscribe" {
			t.Fatalf("path = %s, want /subscribe", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: {\"items\":[{\"nodeId\":\"ns=2;s=state\",\"value\":\"running\",\"quality\":\"good\"}]}\n\n")
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var got MetricValue
	reader := OPCUAHTTPReader{}
	err := reader.Subscribe(ctx, endpointFromServer(t, server), []MetricAddress{{Name: "state", NodeID: "ns=2;s=state", Type: "string"}}, 250, func(value MetricValue) {
		got = value
		cancel()
	})
	if err != nil && ctx.Err() == nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	if got.Name != "state" || got.Value != "running" {
		t.Fatalf("unexpected subscription value: %#v", got)
	}
}
