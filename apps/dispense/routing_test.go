package main

import (
	"errors"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"iot-lense-sense/shared/mqttx"
)

type routingToken struct {
	mqtt.Token
	complete bool
	err      error
}

func (t routingToken) WaitTimeout(time.Duration) bool { return t.complete }
func (t routingToken) Error() error                   { return t.err }

type routingClient struct {
	mqttx.Client
	connected bool
	token     routingToken
	topic     string
	payload   any
}

func (c *routingClient) IsConnected() bool { return c.connected }
func (c *routingClient) Publish(topic string, qos byte, retained bool, payload any) mqtt.Token {
	c.topic, c.payload = topic, payload
	return c.token
}

type routingMessage struct {
	mqttx.Message
	payload string
}

func (m routingMessage) Topic() string   { return "source/agent/metrics/temperature" }
func (m routingMessage) Payload() []byte { return []byte(m.payload) }

func resetRoutingState(t *testing.T) {
	t.Helper()
	state = State{Config: Config{TargetPrefix: "target", SourceFilter: "modbus-tcp", BufferLimit: 2}, Metrics: map[string]map[string][]MetricPoint{}, Topics: map[string]bool{}}
	t.Cleanup(func() {
		state = State{Metrics: map[string]map[string][]MetricPoint{}, Topics: map[string]bool{}, Started: time.Now()}
	})
}

func TestMetricRouting(t *testing.T) {
	valid := `{"agentId":"agent","source":{"type":"modbus-tcp"},"metric":{"value":42,"unit":"C"}}`
	for _, tc := range []struct {
		name, payload       string
		connected, complete bool
		err                 error
		forwarded, dropped  int64
	}{
		{"forward", valid, true, true, nil, 1, 0},
		{"invalid JSON", "{", true, true, nil, 0, 1},
		{"filtered", `{"source":{"type":"opcua"}}`, true, true, nil, 0, 0},
		{"disconnected", valid, false, true, nil, 0, 1},
		{"timeout", valid, true, false, nil, 0, 1},
		{"publish error", valid, true, true, errors.New("failed"), 0, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetRoutingState(t)
			client := &routingClient{connected: tc.connected, token: routingToken{complete: tc.complete, err: tc.err}}
			state.OutputClient = client
			onMetric(nil, routingMessage{payload: tc.payload})
			if state.ReceivedCount != 1 || state.ForwardedCount != tc.forwarded || state.DroppedCount != tc.dropped {
				t.Fatalf("counts received=%d forwarded=%d dropped=%d", state.ReceivedCount, state.ForwardedCount, state.DroppedCount)
			}
			if tc.forwarded > 0 && (client.topic != "target/agent/metrics/temperature" || string(client.payload.([]byte)) != valid || state.LastOutput == "") {
				t.Fatal("incorrect forwarded message")
			}
		})
	}
}

func TestMetricBufferLimit(t *testing.T) {
	resetRoutingState(t)
	for _, payload := range []string{`{"agentId":"a","source":{"type":"modbus-tcp"},"metric":{"name":"m","value":1}}`, `{"agentId":"a","source":{"type":"modbus-tcp"},"metric":{"name":"m","value":2}}`, `{"agentId":"a","source":{"type":"modbus-tcp"},"metric":{"name":"m","value":3}}`} {
		onMetric(nil, routingMessage{payload: payload})
	}
	points := state.Metrics["a"]["m"]
	if len(points) != 2 || points[0].Value != 2 || points[1].Value != 3 {
		t.Fatalf("unexpected buffer: %+v", points)
	}
}

func TestNonNumericMetricsAreForwarded(t *testing.T) {
	for _, value := range []string{`"running"`, `true`, `null`} {
		t.Run(value, func(t *testing.T) {
			resetRoutingState(t)
			client := &routingClient{connected: true, token: routingToken{complete: true}}
			state.OutputClient = client
			payload := `{"agentId":"a","source":{"type":"modbus-tcp"},"metric":{"name":"state","value":` + value + `}}`
			onMetric(nil, routingMessage{payload: payload})
			if state.ForwardedCount != 1 || state.DroppedCount != 0 || client.topic != "target/a/metrics/state" || string(client.payload.([]byte)) != payload {
				t.Fatal("non-numeric telemetry was not forwarded unchanged")
			}
			if len(state.Metrics) != 0 {
				t.Fatal("non-numeric telemetry must not appear as a zero-valued chart point")
			}
		})
	}
}

func TestLoadConfigValidation(t *testing.T) {
	for _, raw := range []string{`{"bufferLimit":-1}`, `{"bufferLimit":0}`, `{"instanceId":"partial","bufferLimit":"invalid"}`} {
		t.Run(raw, func(t *testing.T) {
			resetRoutingState(t)
			t.Setenv("DISPENSE_INSTANCE_ID", "original")
			t.Setenv("DISPENSE_CONFIG_JSON", raw)
			config := loadConfig()
			if config.BufferLimit != 1500 || config.InstanceID != "original" {
				t.Fatalf("invalid fallback: %+v", config)
			}
		})
	}
}
