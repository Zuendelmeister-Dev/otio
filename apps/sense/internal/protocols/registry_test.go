package protocols

import "testing"

func TestRegistryContainsBuiltInReaders(t *testing.T) {
	registry := NewRegistry()

	if _, err := registry.Reader("modbus-tcp"); err != nil {
		t.Fatalf("modbus reader missing: %v", err)
	}
	if _, err := registry.Reader("opcua"); err != nil {
		t.Fatalf("opcua reader missing: %v", err)
	}
	if _, err := registry.Subscriber("opcua"); err != nil {
		t.Fatalf("opcua subscriber missing: %v", err)
	}
	if _, err := registry.Subscriber("modbus-tcp"); err == nil {
		t.Fatalf("modbus-tcp should not expose subscriptions")
	}
}
