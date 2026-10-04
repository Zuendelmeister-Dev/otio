package app

import (
	"context"
	"testing"

	"iot-lense-sense/sense/internal/protocols"
	"iot-lense-sense/shared/protocolmeta"
)

type extensionReader struct{}

func (extensionReader) Read(context.Context, protocols.SourceEndpoint, protocols.MetricAddress) (protocols.MetricValue, error) {
	return protocols.MetricValue{}, nil
}
func (extensionReader) Subscribe(context.Context, protocols.SourceEndpoint, []protocols.MetricAddress, int, func(protocols.MetricValue)) error {
	return nil
}

func TestValidateUsesRegisteredSubscriptionCapability(t *testing.T) {
	previous := sourceProtocols
	t.Cleanup(func() { sourceProtocols = previous })
	registry := protocols.NewRegistry()
	registry.RegisterReader("vendor-stream", extensionReader{})
	registry.RegisterSubscriber("vendor-stream", extensionReader{})
	sourceProtocols = registry
	config := validConfig()
	config.Sources[0].Type = "vendor-stream"
	config.Sources[0].ReadMode = "subscription"
	if problems := validateConfig(config); len(problems) != 0 {
		t.Fatalf("registered vendor subscription was rejected: %v", problems)
	}
}

func TestEveryAdvertisedSourcePresetValidates(t *testing.T) {
	for _, preset := range protocolmeta.Sources() {
		t.Run(preset.Type, func(t *testing.T) {
			config := validConfig()
			metric := MetricConfig{Name: "value", NodeID: preset.Metric.NodeID, Address: preset.Metric.Address, Scale: preset.Metric.Scale}
			if preset.Metric.Register != nil {
				metric.Register = *preset.Metric.Register
			}
			config.Sources = []SourceConfig{{AgentID: "test", Type: preset.Type, Host: "device", Port: preset.Port, UnitID: 1, Metrics: []MetricConfig{metric}}}
			if preset.RequiresConnection {
				config.Sources[0].Options = map[string]any{"connection": preset.Connection}
			}
			if problems := validateConfig(config); len(problems) != 0 {
				t.Fatalf("form preset fails backend validation: %v", problems)
			}
		})
	}
}
