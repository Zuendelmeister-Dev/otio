package mqttx

import (
	"context"
	"encoding/json"
	"errors"
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
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return PublishContext(ctx, client, topic, qos, retained, payload)
}

// PublishContext bounds the caller's wait, not the lifetime of the underlying
// Paho operation. Cancellation or timeout therefore leaves delivery uncertain.
func PublishContext(ctx context.Context, client Client, topic string, qos byte, retained bool, payload any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !IsReady(client) {
		return ErrNotReady
	}
	token := client.Publish(topic, qos, retained, payload)
	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return errors.Join(ErrTimeout, ctx.Err())
		}
		return ctx.Err()
	case <-token.Done():
	}
	if err := token.Error(); err != nil {
		return err
	}
	if !IsReady(client) {
		return ErrConnectionLost
	}
	return nil
}
