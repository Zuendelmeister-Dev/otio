package mqttx

import "testing"

func TestSecretCredentials(t *testing.T) {
	t.Setenv("OTIO_MQTT_USERNAME", "ha-user")
	t.Setenv("OTIO_MQTT_PASSWORD", "ha-secret")
	opts := NewClientOptions(BrokerConfig{Host: "broker", Port: 1883, ClientID: "logical-sense"}, Hooks{})
	if opts.Username != "ha-user" || opts.Password != "ha-secret" || opts.ClientID != "logical-sense" {
		t.Fatal("Secret credentials or logical identity not preserved")
	}
}
