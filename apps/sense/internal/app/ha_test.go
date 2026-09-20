package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHARejectsConfigurationMutations(t *testing.T) {
	t.Setenv("OTIO_CONFIG_READ_ONLY", "true")
	for _, handler := range []http.HandlerFunc{apiApplyConfig, apiRollbackConfig} {
		w := httptest.NewRecorder()
		handler(w, httptest.NewRequest(http.MethodPost, "/", nil))
		if w.Code != http.StatusForbidden {
			t.Fatalf("mutation allowed: %d %s", w.Code, w.Body.String())
		}
	}
}
