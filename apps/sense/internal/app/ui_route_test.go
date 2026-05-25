package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStaticChartAssetIsServedAsJavaScript(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "standard-chart.js"), []byte("window.IoTStandardChart = {};"), 0644); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(dir))))

	req := httptest.NewRequest(http.MethodGet, "/static/standard-chart.js", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "<!doctype html>") {
		t.Fatalf("static JavaScript endpoint returned HTML")
	}
	if !strings.Contains(rec.Body.String(), "IoTStandardChart") {
		t.Fatalf("unexpected static body: %s", rec.Body.String())
	}
}
