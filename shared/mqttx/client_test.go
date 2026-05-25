package mqttx

import "testing"

func TestBrokerURL(t *testing.T) {
	got := BrokerURL("mqtt", 1883)
	want := "tcp://mqtt:1883"
	if got != want {
		t.Fatalf("BrokerURL() = %s, want %s", got, want)
	}
}

func TestNewClientOptions(t *testing.T) {
	options := NewClientOptions(BrokerConfig{Host: "mqtt", Port: 1883, ClientID: "client-1"}, Hooks{})
	if len(options.Servers) != 1 || options.Servers[0].String() != "tcp://mqtt:1883" {
		t.Fatalf("unexpected broker servers: %#v", options.Servers)
	}
	if options.ClientID != "client-1" {
		t.Fatalf("client id = %s, want client-1", options.ClientID)
	}
	if !options.AutoReconnect {
		t.Fatalf("auto reconnect should be enabled")
	}
	if !options.ConnectRetry {
		t.Fatalf("connect retry should be enabled")
	}
}
