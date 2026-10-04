package protocols

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOPCUAQualityIsPreservedForPollAndSubscription(t *testing.T) {
	for _, tc := range []struct {
		upstream string
		want     QualityStatus
	}{{"Good", QualityGood}, {"BadNoCommunication", QualityBad}, {"UncertainLastUsableValue", QualityUncertain}, {"", QualityUnknown}, {"unrecognized", QualityUnknown}} {
		t.Run(tc.upstream, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/read" {
					fmt.Fprintf(w, `{"nodeId":"node","value":42,"quality":%q}`, tc.upstream)
					return
				}
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprintf(w, "data: {\"items\":[{\"nodeId\":\"node\",\"value\":42,\"quality\":%q}]}\n\n", tc.upstream)
			}))
			defer server.Close()
			reader, endpoint := OPCUAHTTPReader{}, endpointFromServer(t, server)
			metric := MetricAddress{Name: "value", NodeID: "node", Scale: 1}
			value, err := reader.Read(context.Background(), endpoint, metric)
			if err != nil || value.Quality != tc.want {
				t.Fatalf("poll quality=%q, error=%v", value.Quality, err)
			}
			var updates []MetricValue
			err = reader.Subscribe(context.Background(), endpoint, []MetricAddress{metric}, 250, func(value MetricValue) { updates = append(updates, value) })
			if err != nil || len(updates) != 1 || updates[0].Quality != tc.want {
				t.Fatalf("subscription quality lost: %+v, %v", updates, err)
			}
		})
	}
}
