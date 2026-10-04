package protocols

import (
	"context"
	"testing"

	"iot-lense-sense/shared/protocolmeta"
)

type fakeReader struct{}

func (fakeReader) Read(ctx context.Context, endpoint SourceEndpoint, metric MetricAddress) (MetricValue, error) {
	return MetricValue{Name: metric.Name, Raw: "raw", Value: "value"}, nil
}

func TestRegistryMatchesSharedCapabilities(t *testing.T) {
	registry := NewRegistry()
	for _, source := range protocolmeta.Sources() {
		if _, err := registry.Reader(source.Type); err != nil {
			t.Fatalf("advertised source %s lacks a reader: %v", source.Type, err)
		}
		_, err := registry.Subscriber(source.Type)
		if source.Subscription != (err == nil) {
			t.Fatalf("subscription capability disagrees with registry for %s", source.Type)
		}
	}
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
