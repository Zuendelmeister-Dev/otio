package protocols

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLabReader(t *testing.T) {
	for _, tc := range []struct {
		body   string
		status int
		want   any
	}{{`{"value":2350}`, 200, 23.5}, {`{"value":"running"}`, 200, "running"}, {`{"error":"offline"}`, 502, nil}, {`not json`, 200, nil}} {
		t.Run(tc.body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]string
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body["protocol"] != "modbus-tcp" || body["address"] != "holding-register:1:UINT" {
					t.Errorf("bad request %v", body)
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			result, err := (LabReader{}).Read(context.Background(), SourceEndpoint{Type: "lab-modbus-tcp", Options: map[string]any{"gatewayURL": server.URL, "connection": "modbus-tcp://device:502"}}, MetricAddress{Name: "temperature", Address: "holding-register:1:UINT", Scale: 0.01})
			if tc.want == nil {
				if err == nil {
					t.Fatal("expected error")
				}
			} else if err != nil || result.Value != tc.want {
				t.Fatalf("got %+v, %v", result, err)
			}
		})
	}
}
func TestAllLabReadersRegistered(t *testing.T) {
	for _, protocol := range LabProtocolTypes {
		if _, err := NewRegistry().Reader("lab-" + protocol); err != nil {
			t.Fatal(err)
		}
	}
}
