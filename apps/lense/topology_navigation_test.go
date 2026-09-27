package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicTopologyNavigationOnly(t *testing.T) {
	response := httptest.NewRecorder()
	apiPublicTopology(response, httptest.NewRequest(http.MethodGet, "/api/topology", nil))
	if response.Code != http.StatusOK || response.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("navigation response: %d %v", response.Code, response.Header())
	}
	var result LenseTopology
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(response.Body.String(), `"configuration"`) || strings.Contains(response.Body.String(), `"password"`) {
		t.Fatal("navigation metadata must exclude configuration fields")
	}
	denied := httptest.NewRecorder()
	apiPublicTopology(denied, httptest.NewRequest(http.MethodPost, "/api/topology", nil))
	if denied.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status: %d", denied.Code)
	}
}
