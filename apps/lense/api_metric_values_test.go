package main

import (
	"os"
	"strings"
	"testing"
)

func TestAPIMetricValuesEndpointIsRegistered(t *testing.T) {
	raw, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"/api/metric-values"`) {
		t.Fatal("metric values endpoint is not registered")
	}
}
