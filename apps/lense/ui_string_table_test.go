package main

import (
	"os"
	"strings"
	"testing"
)

func TestUnifiedNamespaceUsesTableForNonNumericMetrics(t *testing.T) {
	raw, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, want := range []string{"unsValueTable", "renderValueTable", "valueKind!=='number'", "/api/metric-values"} {
		if !strings.Contains(content, want) {
			t.Fatalf("Unified Namespace preview must contain %s", want)
		}
	}
}

func TestConnectionGraphSupportsHorizontalDragScroll(t *testing.T) {
	raw, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, want := range []string{"bindDragScroll", "graph-scroll", "boxWidth=2300", "mousedown"} {
		if !strings.Contains(content, want) {
			t.Fatalf("Connection graph must contain %s", want)
		}
	}
}
