package mqttx

import (
	"encoding/json"
	"time"
)

// MarshalJSON marshals a payload for transport.
func MarshalJSON(payload any) ([]byte, error) {
	return json.Marshal(payload)
}

// PublishJSON marshals and publishes JSON to the given topic.
func PublishJSON(client Client, topic string, qos byte, retained bool, payload any, timeout time.Duration) error {
	raw, err := MarshalJSON(payload)
	if err != nil {
		return err
	}
	return Publish(client, topic, qos, retained, raw, timeout)
}

// Publish sends a raw payload to the given topic.
func Publish(client Client, topic string, qos byte, retained bool, payload any, timeout time.Duration) error {
	token := client.Publish(topic, qos, retained, payload)
	if !token.WaitTimeout(timeout) {
		return ErrTimeout
	}
	return token.Error()
}
