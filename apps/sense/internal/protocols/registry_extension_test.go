package protocols

import (
	"context"
	"testing"
)

type fakeReader struct{}

func (fakeReader) Read(ctx context.Context, endpoint SourceEndpoint, metric MetricAddress) (MetricValue, error) {
	return MetricValue{Name: metric.Name, Raw: "raw", Value: "value"}, nil
}

func TestRegistryAllowsCustomReaderRegistration(t *testing.T) {
	registry := Registry{readers: map[string]Reader{}, subscribers: map[string]Subscriber{}}
	registry.RegisterReader("vendor-protocol", fakeReader{})

	reader, err := registry.Reader("vendor-protocol")
	if err != nil {
		t.Fatalf("custom reader lookup failed: %v", err)
	}

	value, err := reader.Read(context.Background(), SourceEndpoint{AgentID: "machine-01"}, MetricAddress{Name: "state"})
	if err != nil {
		t.Fatalf("custom reader failed: %v", err)
	}
	if value.Name != "state" || value.Value != "value" {
		t.Fatalf("unexpected value: %#v", value)
	}
}
