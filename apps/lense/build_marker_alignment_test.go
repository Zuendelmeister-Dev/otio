package main

import (
	"os"
	"strings"
	"testing"
)

func TestLenseBuildMarkerTestsAreAligned(t *testing.T) {
	htmlRaw, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}

	testRaw, err := os.ReadFile("lense_self_routing_test.go")
	if err != nil {
		t.Fatal(err)
	}

	html := string(htmlRaw)
	test := string(testRaw)
	marker := "const LENSE_MENU_BUILD_MARKER='v65-presense-simulation-vs-external-source'"

	if !strings.Contains(html, marker) {
		t.Fatalf("static index.html does not contain expected marker %s", marker)
	}
	if !strings.Contains(test, marker) {
		t.Fatalf("lense_self_routing_test.go does not contain expected marker %s", marker)
	}
	if strings.Contains(test, "v64-lense-self-routing-tests-fixed") {
		t.Fatalf("stale v64 marker must not remain in lense_self_routing_test.go")
	}
}
